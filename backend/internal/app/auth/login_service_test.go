package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
	"golang.org/x/crypto/bcrypt"
)

// --- fakes ---

type fakeLoginRepo struct {
	user      *domain.User
	findErr   error
	updateErr error
	updated   bool
}

func (f *fakeLoginRepo) FindByUsernameOrEmail(_ context.Context, _ string) (*domain.User, error) {
	return f.user, f.findErr
}

func (f *fakeLoginRepo) UpdateLastLogin(_ context.Context, _ uint, _ int, _ time.Time) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = true
	return nil
}

type fakeRateLimitRepo struct {
	loginAttemptCount int64
	failedCount       int64
	lockoutExpiry     time.Time
	incrLoginErr      error
	incrFailedErr     error
}

func (f *fakeRateLimitRepo) IncrLoginAttempt(_ context.Context, _ string) (int64, error) {
	if f.incrLoginErr != nil {
		return 0, f.incrLoginErr
	}
	f.loginAttemptCount++
	return f.loginAttemptCount, nil
}

func (f *fakeRateLimitRepo) IncrFailedLogin(_ context.Context, _ string) (int64, error) {
	if f.incrFailedErr != nil {
		return 0, f.incrFailedErr
	}
	f.failedCount++
	return f.failedCount, nil
}

func (f *fakeRateLimitRepo) ResetFailedLogin(_ context.Context, _ string) error { return nil }

func (f *fakeRateLimitRepo) GetFailedLoginCount(_ context.Context, _ string) (int64, error) {
	return f.failedCount, nil
}

func (f *fakeRateLimitRepo) SetLockout(_ context.Context, _ string) (time.Time, error) {
	f.lockoutExpiry = time.Now().Add(15 * time.Minute)
	return f.lockoutExpiry, nil
}

func (f *fakeRateLimitRepo) GetLockoutExpiry(_ context.Context, _ string) (time.Time, error) {
	return f.lockoutExpiry, nil
}

type fakeRefreshRepo struct {
	storeErr error
}

func (f *fakeRefreshRepo) Store(_ context.Context, _ string, _ uint) error {
	return f.storeErr
}
func (f *fakeRefreshRepo) GetUserID(_ context.Context, _ string) (uint, bool, error) {
	return 0, false, nil
}
func (f *fakeRefreshRepo) Delete(_ context.Context, _ string) (bool, error) { return true, nil }
func (f *fakeRefreshRepo) DeleteAllForUser(_ context.Context, _ uint) error  { return nil }

// --- helpers ---

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	return key
}

func activeUser() *domain.User {
	hash, err := bcrypt.GenerateFromPassword([]byte("StrongP@ss1!"), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return &domain.User{
		ID:         1,
		Username:   "johndoe",
		Email:      "john@example.com",
		Password:   string(hash),
		Active:     true,
		LoginCount: 0,
	}
}

// --- tests ---

func TestLoginService_HappyPath(t *testing.T) {
	key := generateTestKey(t)
	loginRepo := &fakeLoginRepo{user: activeUser()}
	rateRepo := &fakeRateLimitRepo{}
	svc := svcauth.NewLoginService(loginRepo, rateRepo, &fakeRefreshRepo{}, key)

	resp, err := svc.Login(context.Background(), "127.0.0.1", domain.LoginRequest{
		Username: "johndoe",
		Password: "StrongP@ss1!",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token, got empty string")
	}
	if resp.RefreshToken == "" {
		t.Error("expected refresh token, got empty string")
	}
	if !loginRepo.updated {
		t.Error("expected UpdateLastLogin to be called")
	}
}

func TestLoginService_UserNotFound_ReturnsInvalidCredentials(t *testing.T) {
	key := generateTestKey(t)
	loginRepo := &fakeLoginRepo{user: nil}
	rateRepo := &fakeRateLimitRepo{}
	svc := svcauth.NewLoginService(loginRepo, rateRepo, &fakeRefreshRepo{}, key)

	_, err := svc.Login(context.Background(), "127.0.0.1", domain.LoginRequest{
		Username: "nobody",
		Password: "anything",
	})

	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginService_WrongPassword_ReturnsInvalidCredentials(t *testing.T) {
	key := generateTestKey(t)
	loginRepo := &fakeLoginRepo{user: activeUser()}
	rateRepo := &fakeRateLimitRepo{}
	svc := svcauth.NewLoginService(loginRepo, rateRepo, &fakeRefreshRepo{}, key)

	_, err := svc.Login(context.Background(), "127.0.0.1", domain.LoginRequest{
		Username: "johndoe",
		Password: "wrongpassword",
	})

	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginService_InactiveAccount_ReturnsForbidden(t *testing.T) {
	key := generateTestKey(t)
	u := activeUser()
	u.Active = false
	loginRepo := &fakeLoginRepo{user: u}
	rateRepo := &fakeRateLimitRepo{}
	svc := svcauth.NewLoginService(loginRepo, rateRepo, &fakeRefreshRepo{}, key)

	_, err := svc.Login(context.Background(), "127.0.0.1", domain.LoginRequest{
		Username: "johndoe",
		Password: "StrongP@ss1!",
	})

	if !errors.Is(err, domain.ErrAccountInactive) {
		t.Errorf("expected ErrAccountInactive, got %v", err)
	}
}

func TestLoginService_AlreadyLockedAccount_ReturnsLocked(t *testing.T) {
	key := generateTestKey(t)
	loginRepo := &fakeLoginRepo{user: activeUser()}
	rateRepo := &fakeRateLimitRepo{
		failedCount:   5,
		lockoutExpiry: time.Now().Add(10 * time.Minute),
	}
	svc := svcauth.NewLoginService(loginRepo, rateRepo, &fakeRefreshRepo{}, key)

	_, err := svc.Login(context.Background(), "127.0.0.1", domain.LoginRequest{
		Username: "johndoe",
		Password: "StrongP@ss1!",
	})

	var lockErr svcauth.ErrLocked
	if !errors.As(err, &lockErr) {
		t.Errorf("expected ErrLocked, got %T: %v", err, err)
	}
}

func TestLoginService_FifthFailure_TriggersLockout(t *testing.T) {
	key := generateTestKey(t)
	loginRepo := &fakeLoginRepo{user: activeUser()}
	rateRepo := &fakeRateLimitRepo{failedCount: 4} // 4 prior failures; this attempt is the 5th
	svc := svcauth.NewLoginService(loginRepo, rateRepo, &fakeRefreshRepo{}, key)

	_, err := svc.Login(context.Background(), "127.0.0.1", domain.LoginRequest{
		Username: "johndoe",
		Password: "wrongpassword",
	})

	var lockErr svcauth.ErrLocked
	if !errors.As(err, &lockErr) {
		t.Errorf("expected ErrLocked on 5th failure, got %T: %v", err, err)
	}
}

func TestLoginService_RateLimitExceeded_ReturnsRateLimited(t *testing.T) {
	key := generateTestKey(t)
	loginRepo := &fakeLoginRepo{user: activeUser()}
	rateRepo := &fakeRateLimitRepo{loginAttemptCount: 20} // already at 20; next incr → 21
	svc := svcauth.NewLoginService(loginRepo, rateRepo, &fakeRefreshRepo{}, key)

	_, err := svc.Login(context.Background(), "127.0.0.1", domain.LoginRequest{
		Username: "johndoe",
		Password: "StrongP@ss1!",
	})

	if !errors.Is(err, domain.ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}
