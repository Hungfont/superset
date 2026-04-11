package middleware

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const UserContextKey = "user"

// JWTMiddleware validates RS256 Bearer tokens, checks the jti blacklist,
// loads the user from cache (falling back to DB), and injects UserContext.
// Returns 401 for all token failures and 403 for deactivated accounts.
func JWTMiddleware(pubKey *rsa.PublicKey, jwtRepo domain.JWTRepository, userRepo domain.UserRepository) gin.HandlerFunc {
	keyFunc := func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	}

	return func(c *gin.Context) {
		// 1. Extract Bearer token
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenMissing.Error()})
			return
		}
		raw := strings.TrimPrefix(header, "Bearer ")

		// 2. Verify RS256 signature + standard claims (exp, iat)
		token, err := jwt.Parse(raw, keyFunc, jwt.WithValidMethods([]string{"RS256"}))
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
			return
		}

		jti, _ := claims["jti"].(string)
		sub, _ := claims["sub"].(string)

		// 3. Check jti blacklist
		if jti != "" {
			revoked, err := jwtRepo.IsBlacklisted(c.Request.Context(), jti)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}
			if revoked {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenRevoked.Error()})
				return
			}
		}

		// 4. Parse user ID from sub claim
		uid, err := strconv.ParseUint(sub, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
			return
		}
		userID := uint(uid)

		// 5. Load user from Redis cache; fall back to DB on miss
		uctx, err := resolveUser(c, userID, jwtRepo, userRepo)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if uctx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrTokenInvalid.Error()})
			return
		}

		// 6. Reject deactivated users
		if !uctx.Active {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrAccountInactive.Error()})
			return
		}

		// 7. Inject UserContext and proceed
		c.Set(UserContextKey, *uctx)
		c.Next()
	}
}

// resolveUser returns the UserContext from Redis cache or DB (with cache repopulation).
func resolveUser(c *gin.Context, userID uint, jwtRepo domain.JWTRepository, userRepo domain.UserRepository) (*domain.UserContext, error) {
	// Try cache first
	cached, err := jwtRepo.GetCachedUser(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return cached, nil
	}

	// Cache miss — query DB
	user, err := userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	uctx := &domain.UserContext{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Active:   user.Active,
	}

	// Repopulate cache (best-effort; errors are non-fatal)
	_ = jwtRepo.SetCachedUser(c.Request.Context(), userID, uctx)

	return uctx, nil
}
