package auth

import (
	"context"
	"time"

	domain "superset/auth-service/internal/domain/auth"
)

// LogoutService revokes access/refresh tokens and supports logout-all behavior.
// The operation is intentionally best-effort and idempotent.
type LogoutService struct {
	jwtRepo     domain.JWTRepository
	refreshRepo domain.RefreshRepository
}

func NewLogoutService(jwtRepo domain.JWTRepository, refreshRepo domain.RefreshRepository) *LogoutService {
	return &LogoutService{jwtRepo: jwtRepo, refreshRepo: refreshRepo}
}

// Logout revokes the current access token jti (for its remaining lifetime) and
// refresh tokens according to request mode. All repository errors are ignored to
// preserve idempotent logout semantics.
func (s *LogoutService) Logout(ctx context.Context, req domain.LogoutRequest) {
	s.blacklistAccessToken(ctx, req.JTI, req.AccessTokenExpiresAt)

	if req.LogoutAll {
		if req.UserID > 0 {
			_ = s.refreshRepo.DeleteAllForUser(ctx, req.UserID)
		}
		return
	}

	if req.RefreshToken != "" {
		_, _ = s.refreshRepo.Delete(ctx, req.RefreshToken)
	}
}

func (s *LogoutService) blacklistAccessToken(ctx context.Context, jti string, expiresAt time.Time) {
	if jti == "" || expiresAt.IsZero() {
		return
	}
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return
	}
	_ = s.jwtRepo.BlacklistJTI(ctx, jti, ttl)
}
