package query

import (
	"context"
	"crypto/rsa"
	"log"
	"net/http"
	"strings"
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

var defaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		allowedOrigins := []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"http://localhost:8080",
			"https://localhost:5173",
			"https://localhost:3000",
			"https://localhost:8080",
		}
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
		return strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "https://localhost:")
	},
}

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
		upgrader:  defaultUpgrader,
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

	ctx, cancel := context.WithCancel(context.Background())
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

func isAdminRole(roles []string) bool {
	for _, role := range roles {
		if role == "Admin" {
			return true
		}
	}
	return false
}
