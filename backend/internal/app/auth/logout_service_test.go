package auth_test

import (
	"context"
	"testing"
	"time"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
)

type fakeJWTRepoForLogout struct {
	jti string
	ttl time.Duration
}

func (f *fakeJWTRepoForLogout) IsBlacklisted(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (f *fakeJWTRepoForLogout) GetCachedUser(_ context.Context, _ uint) (*domain.UserContext, error) {
	return nil, nil
}
func (f *fakeJWTRepoForLogout) SetCachedUser(_ context.Context, _ uint, _ *domain.UserContext) error {
	return nil
}
func (f *fakeJWTRepoForLogout) BlacklistJTI(_ context.Context, jti string, ttl time.Duration) error {
	f.jti = jti
	f.ttl = ttl
	return nil
}

type fakeRefreshRepoForLogout struct {
	deletedToken    string
	deleteAllUserID uint
}

func (f *fakeRefreshRepoForLogout) Store(_ context.Context, _ string, _ uint) error { return nil }
func (f *fakeRefreshRepoForLogout) GetUserID(_ context.Context, _ string) (uint, bool, error) {
	return 0, false, nil
}
func (f *fakeRefreshRepoForLogout) Delete(_ context.Context, token string) (bool, error) {
	f.deletedToken = token
	return true, nil
}
func (f *fakeRefreshRepoForLogout) DeleteAllForUser(_ context.Context, userID uint) error {
	f.deleteAllUserID = userID
	return nil
}

func TestLogoutService_SingleSession_BlacklistsAndDeletesRefreshToken(t *testing.T) {
	jwtRepo := &fakeJWTRepoForLogout{}
	refreshRepo := &fakeRefreshRepoForLogout{}
	svc := svcauth.NewLogoutService(jwtRepo, refreshRepo)

	req := domain.LogoutRequest{
		UserID:               42,
		JTI:                  "jti-123",
		AccessTokenExpiresAt: time.Now().Add(10 * time.Minute),
		RefreshToken:         "refresh-abc",
		LogoutAll:            false,
	}

	svc.Logout(context.Background(), req)

	if jwtRepo.jti != "jti-123" {
		t.Fatalf("expected jti to be blacklisted, got %q", jwtRepo.jti)
	}
	if jwtRepo.ttl <= 0 {
		t.Fatalf("expected positive ttl, got %v", jwtRepo.ttl)
	}
	if refreshRepo.deletedToken != "refresh-abc" {
		t.Fatalf("expected refresh token delete for current session, got %q", refreshRepo.deletedToken)
	}
	if refreshRepo.deleteAllUserID != 0 {
		t.Fatalf("did not expect logout-all delete, got userID=%d", refreshRepo.deleteAllUserID)
	}
}

func TestLogoutService_AllDevices_BlacklistsAndDeletesAllUserSessions(t *testing.T) {
	jwtRepo := &fakeJWTRepoForLogout{}
	refreshRepo := &fakeRefreshRepoForLogout{}
	svc := svcauth.NewLogoutService(jwtRepo, refreshRepo)

	req := domain.LogoutRequest{
		UserID:               7,
		JTI:                  "jti-xyz",
		AccessTokenExpiresAt: time.Now().Add(15 * time.Minute),
		RefreshToken:         "refresh-current",
		LogoutAll:            true,
	}

	svc.Logout(context.Background(), req)

	if jwtRepo.jti != "jti-xyz" {
		t.Fatalf("expected jti to be blacklisted, got %q", jwtRepo.jti)
	}
	if refreshRepo.deleteAllUserID != 7 {
		t.Fatalf("expected delete all sessions for user 7, got %d", refreshRepo.deleteAllUserID)
	}
	if refreshRepo.deletedToken != "" {
		t.Fatalf("did not expect single-token delete on logout-all, got %q", refreshRepo.deletedToken)
	}
}

func TestLogoutService_Idempotent_NoTokensStillReturns(t *testing.T) {
	jwtRepo := &fakeJWTRepoForLogout{}
	refreshRepo := &fakeRefreshRepoForLogout{}
	svc := svcauth.NewLogoutService(jwtRepo, refreshRepo)

	svc.Logout(context.Background(), domain.LogoutRequest{})

	if jwtRepo.jti != "" {
		t.Fatalf("did not expect blacklist when claims are missing, got %q", jwtRepo.jti)
	}
	if refreshRepo.deletedToken != "" || refreshRepo.deleteAllUserID != 0 {
		t.Fatalf("did not expect refresh revocation calls, got deleteToken=%q deleteAllUserID=%d", refreshRepo.deletedToken, refreshRepo.deleteAllUserID)
	}
}
