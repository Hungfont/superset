package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "superset/auth-service/internal/domain/auth"
)

// ErrInvalidHash is returned when the hash does not match any pending registration.
var ErrInvalidHash = fmt.Errorf("invalid or already used verification link")

// ErrExpiredHash is returned when the registration link is older than 24 hours.
var ErrExpiredHash = fmt.Errorf("verification link has expired")

// VerifyService handles email verification and account activation.
type VerifyService struct {
	repo domain.VerifyRepository
}

func NewVerifyService(repo domain.VerifyRepository) *VerifyService {
	return &VerifyService{repo: repo}
}

// Verify looks up the hash, checks expiry, and activates the account.
func (s *VerifyService) Verify(ctx context.Context, hash string) error {
	reg, err := s.repo.FindByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("looking up hash: %w", err)
	}
	if reg == nil {
		return ErrInvalidHash
	}

	if time.Since(reg.CreatedAt) > 24*time.Hour {
		return ErrExpiredHash
	}

	if err := s.repo.Activate(ctx, reg); err != nil {
		if errors.Is(err, domain.ErrAlreadyActivated) {
			return ErrInvalidHash // concurrent duplicate → treat as "already used" → 404
		}
		return fmt.Errorf("activating account: %w", err)
	}

	return nil
}
