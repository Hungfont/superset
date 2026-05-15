package query

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	svcquery "superset/auth-service/internal/app/query"
	domain "superset/auth-service/internal/domain/auth"
	domainquery "superset/auth-service/internal/domain/query"
	"superset/auth-service/internal/delivery/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	wsHeartbeatInterval = 30 * time.Second
	wsHeartbeatTimeout  = 35 * time.Second
	wsWriteTimeout      = 10 * time.Second
	wsReadTimeout       = 35 * time.Second
)

type WSHandler struct {
	pubKey    *rsa.PublicKey
	jwtRepo   domain.JWTRepository
	userRepo  domain.UserRepository
	rlsRepo   svcquery.RoleNameProvider
	queryRepo domainquery.Repository
	rdb       *redis.Client
	upgrader  websocket.Upgrader
}

func NewWSHandler(
	pubKey *rsa.PublicKey,
	jwtRepo domain.JWTRepository,
	userRepo domain.UserRepository,
	rlsRepo svcquery.RoleNameProvider,
	queryRepo domainquery.Repository,
	rdb *redis.Client,
) *WSHandler {
	return &WSHandler{
		pubKey:    pubKey,
		jwtRepo:   jwtRepo,
		userRepo:  userRepo,
		rlsRepo:   rlsRepo,
		queryRepo: queryRepo,
		rdb:       rdb,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			// Accept all origins — authentication is handled via JWT in query string,
			// so the origin check is redundant and would break non-localhost deployments.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (h *WSHandler) Handle(c *gin.Context) {
	queryID := c.Param("query_id")
	if queryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query_id required"})
		return
	}

	tokenStr := c.Query("token")

	userCtx, statusCode := middleware.ValidateJWTFromQuery(tokenStr, h.pubKey, h.jwtRepo, h.userRepo)
	if statusCode != 0 {
		c.AbortWithStatus(statusCode)
		return
	}

	q, err := h.queryRepo.GetByID(c.Request.Context(), queryID)
	if err != nil {
		log.Printf("[ws] error fetching query %s: %v", queryID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "query not found"})
		return
	}
	if q == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "query not found"})
		return
	}

	if q.UserID != userCtx.ID {
		roles, err := h.rlsRepo.GetRoleNamesByUser(c.Request.Context(), userCtx.ID)
		if err != nil || !isAdminRole(roles) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ws] upgrade error for query %s: %v", queryID, err)
		return
	}

	// Derive context from the Gin request so cancellation is tied to the HTTP
	// request lifecycle (client disconnect propagates to all child goroutines).
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	redisSub := h.rdb.Subscribe(ctx, "query:status:"+queryID)
	defer redisSub.Close()

	var wg sync.WaitGroup
	closed := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		ch := redisSub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if err := conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
					log.Printf("[ws] write error for query %s: %v", queryID, err)
					return
				}
			case <-closed:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(wsHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
					return
				}
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("[ws] heartbeat error for query %s: %v", queryID, err)
					return
				}
			case <-closed:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
		return nil
	})

	// Send current query state — if the worker finished before the WebSocket
	// connected (Redis Pub/Sub drops pre-subscription messages), the client
	// would otherwise never learn the result.
	h.sendQueryState(conn, queryID)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	close(closed)
	cancel()
	conn.Close()
	wg.Wait()
}

func (h *WSHandler) sendQueryState(conn *websocket.Conn, queryID string) {
	q, err := h.queryRepo.GetByID(context.Background(), queryID)
	if err != nil || q == nil {
		return
	}

	switch q.Status {
	case "success":
		// Query already completed — try to send result
		if q.ResultsKey != "" && h.rdb != nil {
			resultJSON, rerr := h.rdb.Get(context.Background(), q.ResultsKey).Bytes()
			if rerr == nil {
				var result svcquery.ExecuteResponse
				if json.Unmarshal(resultJSON, &result) == nil {
					data := map[string]interface{}{
						"rows":    result.Data,
						"columns": result.Columns,
					}
					event := map[string]interface{}{
						"type":     "done",
						"query_id": queryID,
						"data":     data,
					}
					if eventJSON, jerr := json.Marshal(event); jerr == nil {
						conn.WriteMessage(websocket.TextMessage, eventJSON)
					}
					return
				}
			}
		}
		// Fallback: send result_ready so client can poll
		event := map[string]interface{}{
			"type":         "result_ready",
			"query_id":     queryID,
			"download_url": "/api/v1/query/" + queryID + "/result/download",
		}
		if eventJSON, jerr := json.Marshal(event); jerr == nil {
			conn.WriteMessage(websocket.TextMessage, eventJSON)
		}

	case "failed", "stopped":
		msg := q.ErrorMessage
		if msg == "" {
			msg = "Query " + q.Status
		}
		event := map[string]interface{}{
			"type":     "error",
			"query_id": queryID,
			"message":  msg,
		}
		if eventJSON, jerr := json.Marshal(event); jerr == nil {
			conn.WriteMessage(websocket.TextMessage, eventJSON)
		}

	case "running", "pending":
		pct := 50
		if q.Status == "pending" {
			pct = 10
		}
		event := map[string]interface{}{
			"type":     "progress",
			"query_id": queryID,
			"progress": q.Status,
			"percent":  pct,
		}
		if eventJSON, jerr := json.Marshal(event); jerr == nil {
			conn.WriteMessage(websocket.TextMessage, eventJSON)
		}
	}
}

func isAdminRole(roles []string) bool {
	for _, role := range roles {
		if role == "Admin" {
			return true
		}
	}
	return false
}
