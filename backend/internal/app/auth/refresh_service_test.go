package auth_test

import (
	"context"
	"errors"
	"testing"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
)

// --- fakes ---

type fakeRefreshRepoForSvc struct {
	// GetUserID controls
	storedUserID uint
	found        bool
	getUserErr   error
	// Delete controls
	deleted   bool
	deleteErr error
	// DeleteAllForUser tracking
	deletedAll bool
	deleteAllErr error
}

func (f *fakeRefreshRepoForSvc) Store(_ context.Context, _ string, _ uint) error { return nil }

func (f *fakeRefreshRepoForSvc) GetUserID(_ context.Context, _ string) (uint, bool, error) {
	return f.storedUserID, f.found, f.getUserErr
}

func (f *fakeRefreshRepoForSvc) Delete(_ context.Context, _ string) (bool, error) {
	return f.deleted, f.deleteErr
}

func (f *fakeRefreshRepoForSvc) DeleteAllForUser(_ context.Context, _ uint) error {
	f.deletedAll = true
	return f.deleteAllErr
}

type fakeUserRepoForRefresh struct {
	user    *domain.User
	findErr error
}

func (f *fakeUserRepoForRefresh) FindByID(_ context.Context, _ uint) (*domain.User, error) {
	return f.user, f.findErr
}

// --- helpers ---

func activeRefreshUser() *domain.User {
	return &domain.User{
		ID:       42,
		Username: "alice",
		Email:    "alice@example.com",
		Active:   true,
	}
}

// --- tests ---

func TestRefreshService_HappyPath_RotatesToken(t *testing.T) {
	key := generateTestKey(t)
	refreshRepo := &fakeRefreshRepoForSvc{storedUserID: 42, found: true, deleted: true}
	userRepo := &fakeUserRepoForRefresh{user: activeRefreshUser()}
	svc := svcauth.NewRefreshService(refreshRepo, userRepo, key)

	resp, err := svc.Refresh(context.Background(), "old-token")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token, got empty string")
	}
	if resp.RefreshToken == "" {
		t.Error("expected new refresh token, got empty string")
	}
}

func TestRefreshService_UnknownToken_ReturnsTokenInvalid(t *testing.T) {
	key := generateTestKey(t)
	refreshRepo := &fakeRefreshRepoForSvc{found: false}
	userRepo := &fakeUserRepoForRefresh{}
	svc := svcauth.NewRefreshService(refreshRepo, userRepo, key)

	_, err := svc.Refresh(context.Background(), "unknown-token")

	if !errors.Is(err, domain.ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestRefreshService_ReuseDetected_InvalidatesAllSessions(t *testing.T) {
	key := generateTestKey(t)
	// Token is found (GET returns userID) but Delete returns false (already deleted = reuse).
	refreshRepo := &fakeRefreshRepoForSvc{storedUserID: 42, found: true, deleted: false}
	userRepo := &fakeUserRepoForRefresh{}
	svc := svcauth.NewRefreshService(refreshRepo, userRepo, key)

	_, err := svc.Refresh(context.Background(), "reused-token")

	if !errors.Is(err, domain.ErrTokenReused) {
		t.Errorf("expected ErrTokenReused, got %v", err)
	}
	if !refreshRepo.deletedAll {
		t.Error("expected DeleteAllForUser to be called on reuse attack")
	}
}

func TestRefreshService_InactiveUser_ReturnsAccountInactive(t *testing.T) {
	key := generateTestKey(t)
	refreshRepo := &fakeRefreshRepoForSvc{storedUserID: 42, found: true, deleted: true}
	u := activeRefreshUser()
	u.Active = false
	userRepo := &fakeUserRepoForRefresh{user: u}
	svc := svcauth.NewRefreshService(refreshRepo, userRepo, key)

	_, err := svc.Refresh(context.Background(), "valid-token")

	if !errors.Is(err, domain.ErrAccountInactive) {
		t.Errorf("expected ErrAccountInactive, got %v", err)
	}
}

func TestRefreshService_UserNotFound_ReturnsAccountInactive(t *testing.T) {
	key := generateTestKey(t)
	refreshRepo := &fakeRefreshRepoForSvc{storedUserID: 42, found: true, deleted: true}
	userRepo := &fakeUserRepoForRefresh{user: nil} // DB returns nil
	svc := svcauth.NewRefreshService(refreshRepo, userRepo, key)

	_, err := svc.Refresh(context.Background(), "valid-token")

	if !errors.Is(err, domain.ErrAccountInactive) {
		t.Errorf("expected ErrAccountInactive for missing user, got %v", err)
	}
}

func TestRefreshService_RepoError_ReturnsError(t *testing.T) {
	key := generateTestKey(t)
	repoErr := errors.New("redis down")
	refreshRepo := &fakeRefreshRepoForSvc{getUserErr: repoErr}
	userRepo := &fakeUserRepoForRefresh{}
	svc := svcauth.NewRefreshService(refreshRepo, userRepo, key)

	_, err := svc.Refresh(context.Background(), "any-token")

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
