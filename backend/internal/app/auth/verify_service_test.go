package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
)

// --- fakes ---

type fakeVerifyRepo struct {
	reg        *domain.RegisterUser
	findErr    error
	activateErr error
	activated  bool
}

func (f *fakeVerifyRepo) FindByHash(_ context.Context, _ string) (*domain.RegisterUser, error) {
	return f.reg, f.findErr
}

func (f *fakeVerifyRepo) Activate(_ context.Context, _ *domain.RegisterUser) error {
	if f.activateErr != nil {
		return f.activateErr
	}
	f.activated = true
	return nil
}

func pendingReg(createdAt time.Time) *domain.RegisterUser {
	return &domain.RegisterUser{
		ID:               1,
		FirstName:        "Jane",
		LastName:         "Doe",
		Username:         "janedoe",
		Email:            "jane@example.com",
		Password:         "$2a$12$hashedpassword",
		RegistrationHash: "abc123",
		CreatedAt:        createdAt,
	}
}

// --- tests ---

func TestVerifyService_HappyPath(t *testing.T) {
	repo := &fakeVerifyRepo{reg: pendingReg(time.Now().Add(-1 * time.Hour))}
	svc := svcauth.NewVerifyService(repo)

	if err := svc.Verify(context.Background(), "abc123"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.activated {
		t.Error("expected Activate to be called")
	}
}

func TestVerifyService_InvalidHash(t *testing.T) {
	repo := &fakeVerifyRepo{reg: nil}
	svc := svcauth.NewVerifyService(repo)

	err := svc.Verify(context.Background(), "badHash")

	if !errors.Is(err, svcauth.ErrInvalidHash) {
		t.Errorf("expected ErrInvalidHash, got %v", err)
	}
}

func TestVerifyService_ExpiredHash(t *testing.T) {
	repo := &fakeVerifyRepo{reg: pendingReg(time.Now().Add(-25 * time.Hour))}
	svc := svcauth.NewVerifyService(repo)

	err := svc.Verify(context.Background(), "abc123")

	if !errors.Is(err, svcauth.ErrExpiredHash) {
		t.Errorf("expected ErrExpiredHash, got %v", err)
	}
}

func TestVerifyService_FindError(t *testing.T) {
	repo := &fakeVerifyRepo{findErr: errors.New("db error")}
	svc := svcauth.NewVerifyService(repo)

	err := svc.Verify(context.Background(), "abc123")

	if err == nil {
		t.Fatal("expected error from FindByHash, got nil")
	}
	if errors.Is(err, svcauth.ErrInvalidHash) || errors.Is(err, svcauth.ErrExpiredHash) {
		t.Errorf("expected wrapped db error, got sentinel: %v", err)
	}
}

func TestVerifyService_ActivateError(t *testing.T) {
	repo := &fakeVerifyRepo{
		reg:         pendingReg(time.Now().Add(-1 * time.Hour)),
		activateErr: errors.New("tx failed"),
	}
	svc := svcauth.NewVerifyService(repo)

	err := svc.Verify(context.Background(), "abc123")

	if err == nil {
		t.Fatal("expected error from Activate, got nil")
	}
}
