package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	domain "superset/auth-service/internal/domain/auth"
)

// RefreshService implements refresh token rotation (AUTH-005).
// On each call it:
//  1. Validates the incoming refresh token exists in Redis.
//  2. Atomically deletes it (detects reuse when delete misses).
//  3. Invalidates ALL sessions on reuse-attack.
//  4. Issues a new RS256 access token and a fresh refresh token.
type RefreshService struct {
	refreshRepo domain.RefreshRepository
	userRepo    domain.UserRepository
	privKey     *rsa.PrivateKey
}

func NewRefreshService(
	refreshRepo domain.RefreshRepository,
	userRepo domain.UserRepository,
	privKey *rsa.PrivateKey,
) *RefreshService {
	return &RefreshService{
		refreshRepo: refreshRepo,
		userRepo:    userRepo,
		privKey:     privKey,
	}
}

// Refresh validates the old token, rotates it, and returns new credentials.
func (s *RefreshService) Refresh(ctx context.Context, oldToken string) (domain.LoginResponse, error) {
	// 1. Validate token exists and retrieve the owning userID.
	userID, found, err := s.refreshRepo.GetUserID(ctx, oldToken)
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("looking up refresh token: %w", err)
	}
	if !found {
		return domain.LoginResponse{}, domain.ErrTokenInvalid
	}

	// 2. Atomically delete the token.
	// If Delete returns false the key was already gone between the GET and DEL
	// — a concurrent rotation raced us here, indicating a reuse attack.
	deleted, err := s.refreshRepo.Delete(ctx, oldToken)
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("deleting refresh token: %w", err)
	}
	if !deleted {
		// Reuse detected: invalidate every session belonging to this user.
		_ = s.refreshRepo.DeleteAllForUser(ctx, userID)
		return domain.LoginResponse{}, domain.ErrTokenReused
	}

	// 3. Re-fetch user to verify account is still active.
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("fetching user: %w", err)
	}
	if user == nil || !user.Active {
		return domain.LoginResponse{}, domain.ErrAccountInactive
	}

	// 4. Issue new RS256 access token.
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("generating access token: %w", err)
	}

	// 5. Generate and store rotated refresh token.
	newRefresh, err := generateRefreshToken()
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("generating refresh token: %w", err)
	}
	if err := s.refreshRepo.Store(ctx, newRefresh, user.ID); err != nil {
		return domain.LoginResponse{}, fmt.Errorf("storing new refresh token: %w", err)
	}

	return domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
	}, nil
}

func (s *RefreshService) generateAccessToken(user *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   fmt.Sprintf("%d", user.ID),
		"email": user.Email,
		"uname": user.Username,
		"jti":   uuid.NewString(),
		"iat":   now.Unix(),
		"exp":   now.Add(accessTokenExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privKey)
}
