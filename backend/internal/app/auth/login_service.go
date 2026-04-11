package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "superset/auth-service/internal/domain/auth"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	loginRateLimit    = 20
	lockoutThreshold  = 5
	accessTokenExpiry = 15 * time.Minute
)

// ErrLocked carries the expiry time of the account lockout.
type ErrLocked struct {
	Until time.Time
}

func (e ErrLocked) Error() string {
	return fmt.Sprintf("account locked until %s", e.Until.Format(time.RFC3339))
}

func (e ErrLocked) Is(target error) bool {
	return target == domain.ErrAccountLocked
}

// LoginService handles the login business logic.
type LoginService struct {
	loginRepo domain.LoginRepository
	rateRepo  domain.RateLimitRepository
	privKey   *rsa.PrivateKey
}

func NewLoginService(
	loginRepo domain.LoginRepository,
	rateRepo domain.RateLimitRepository,
	privKey *rsa.PrivateKey,
) *LoginService {
	return &LoginService{
		loginRepo: loginRepo,
		rateRepo:  rateRepo,
		privKey:   privKey,
	}
}

// Login validates credentials and returns JWT + refresh token on success.
func (s *LoginService) Login(ctx context.Context, ip string, req domain.LoginRequest) (domain.LoginResponse, error) {
	// 1. Rate limit check
	count, err := s.rateRepo.IncrLoginAttempt(ctx, ip)
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("checking rate limit: %w", err)
	}
	if count > loginRateLimit {
		return domain.LoginResponse{}, domain.ErrRateLimited
	}

	// 2. Find user
	user, err := s.loginRepo.FindByUsernameOrEmail(ctx, req.Username)
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("looking up user: %w", err)
	}
	if user == nil {
		return domain.LoginResponse{}, domain.ErrInvalidCredentials
	}

	// 3. Inactive check
	if !user.Active {
		return domain.LoginResponse{}, domain.ErrAccountInactive
	}

	// 4. Lockout check
	lockedUntil, err := s.rateRepo.GetLockoutExpiry(ctx, user.Username)
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("checking lockout: %w", err)
	}
	if !lockedUntil.IsZero() {
		return domain.LoginResponse{}, ErrLocked{Until: lockedUntil}
	}

	// 5. Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return domain.LoginResponse{}, s.handleFailedAttempt(ctx, user.Username)
	}

	// 6. Reset failed counter on success
	_ = s.rateRepo.ResetFailedLogin(ctx, user.Username)

	// 7. Generate access token (RS256 JWT, 15min)
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("generating access token: %w", err)
	}

	// 8. Generate refresh token (random 32-byte hex, 7d)
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return domain.LoginResponse{}, fmt.Errorf("generating refresh token: %w", err)
	}

	// 9. Store refresh token in Redis
	if err := s.rateRepo.StoreRefreshToken(ctx, refreshToken, user.ID); err != nil {
		return domain.LoginResponse{}, fmt.Errorf("storing refresh token: %w", err)
	}

	// 10. Update login_count and last_login
	now := time.Now()
	if err := s.loginRepo.UpdateLastLogin(ctx, user.ID, user.LoginCount+1, now); err != nil {
		return domain.LoginResponse{}, fmt.Errorf("updating last login: %w", err)
	}

	return domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// handleFailedAttempt increments the failed counter and triggers lockout on the 5th failure.
func (s *LoginService) handleFailedAttempt(ctx context.Context, username string) error {
	count, err := s.rateRepo.IncrFailedLogin(ctx, username)
	if err != nil {
		return domain.ErrInvalidCredentials
	}
	if count >= lockoutThreshold {
		expiry, _ := s.rateRepo.SetLockout(ctx, username)
		return ErrLocked{Until: expiry}
	}
	return domain.ErrInvalidCredentials
}

func (s *LoginService) generateAccessToken(user *domain.User) (string, error) {
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

func generateRefreshToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
