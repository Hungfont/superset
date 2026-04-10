package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	domain "superset/auth-service/internal/domain/auth"
	"superset/auth-service/internal/pkg/email"
	"superset/auth-service/internal/pkg/validator"

	"golang.org/x/crypto/bcrypt"
)

// ErrDuplicateEmail is returned when email is already registered.
var ErrDuplicateEmail = fmt.Errorf("email already registered")

// ErrDuplicateUsername is returned when username is already taken.
var ErrDuplicateUsername = fmt.Errorf("username already taken")

// ErrWeakPassword is returned when the password fails complexity rules.
type ErrWeakPassword struct{ Reason string }

func (e ErrWeakPassword) Error() string { return e.Reason }

// RegisterService handles the user self-registration business logic.
type RegisterService struct {
	repo    domain.RegisterUserRepository
	mailer  email.Sender
	baseURL string
}

func NewRegisterService(repo domain.RegisterUserRepository, mailer email.Sender, baseURL string) *RegisterService {
	return &RegisterService{repo: repo, mailer: mailer, baseURL: baseURL}
}

// Register validates input, persists a pending registration, and fires an
// async verification email. Returns (registrationHash, error).
func (s *RegisterService) Register(ctx context.Context, req domain.RegisterRequest) (string, error) {
	// Password complexity check
	if err := validator.ValidatePasswordComplexity(req.Password); err != nil {
		return "", ErrWeakPassword{Reason: err.Error()}
	}

	// Uniqueness checks
	emailTaken, err := s.repo.EmailExists(ctx, req.Email)
	if err != nil {
		return "", fmt.Errorf("checking email uniqueness: %w", err)
	}
	if emailTaken {
		return "", ErrDuplicateEmail
	}

	usernameTaken, err := s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return "", fmt.Errorf("checking username uniqueness: %w", err)
	}
	if usernameTaken {
		return "", ErrDuplicateUsername
	}

	// Hash password with bcrypt cost=12
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}

	// Generate 64-byte hex registration hash (32 random bytes → 64 hex chars)
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating registration hash: %w", err)
	}
	hash := hex.EncodeToString(raw)

	reg := &domain.RegisterUser{
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Username:         req.Username,
		Email:            req.Email,
		Password:         string(hashed),
		RegistrationHash: hash,
	}

	if err := s.repo.Create(ctx, reg); err != nil {
		return "", fmt.Errorf("persisting registration: %w", err)
	}

	// Send verification email asynchronously
	verificationURL := fmt.Sprintf("%s/auth/verify?hash=%s", s.baseURL, hash)
	go func() {
		_ = s.mailer.SendVerification(req.Email, verificationURL)
	}()

	return hash, nil
}
