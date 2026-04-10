package auth_test

import (
	"context"
	"errors"
	"testing"

	svcauth "superset/auth-service/internal/app/auth"
	domain "superset/auth-service/internal/domain/auth"
)

// --- fakes ---

type fakeRepo struct {
	emailExists    bool
	usernameExists bool
	createErr      error
	created        *domain.RegisterUser
}

func (f *fakeRepo) EmailExists(_ context.Context, _ string) (bool, error) {
	return f.emailExists, nil
}

func (f *fakeRepo) UsernameExists(_ context.Context, _ string) (bool, error) {
	return f.usernameExists, nil
}

func (f *fakeRepo) Create(_ context.Context, r *domain.RegisterUser) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = r
	return nil
}

type fakeMailer struct{ called bool }

func (m *fakeMailer) SendVerification(_, _ string) error {
	m.called = true
	return nil
}

// --- helpers ---

func validRequest() domain.RegisterRequest {
	return domain.RegisterRequest{
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
		Email:     "john@example.com",
		Password:  "StrongP@ss1!",
	}
}

// --- tests ---

func TestRegisterService_HappyPath(t *testing.T) {
	repo := &fakeRepo{}
	mailer := &fakeMailer{}
	svc := svcauth.NewRegisterService(repo, mailer, "http://localhost:3000")

	hash, err := svc.Register(context.Background(), validRequest())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got len=%d", len(hash))
	}
	if repo.created == nil {
		t.Fatal("expected Create to be called")
	}
	if repo.created.RegistrationHash != hash {
		t.Error("stored hash does not match returned hash")
	}
}

func TestRegisterService_WeakPassword(t *testing.T) {
	svc := svcauth.NewRegisterService(&fakeRepo{}, &fakeMailer{}, "http://localhost:3000")
	req := validRequest()
	req.Password = "weak"

	_, err := svc.Register(context.Background(), req)

	var weakErr svcauth.ErrWeakPassword
	if !errors.As(err, &weakErr) {
		t.Errorf("expected ErrWeakPassword, got %T: %v", err, err)
	}
}

func TestRegisterService_DuplicateEmail(t *testing.T) {
	repo := &fakeRepo{emailExists: true}
	svc := svcauth.NewRegisterService(repo, &fakeMailer{}, "http://localhost:3000")

	_, err := svc.Register(context.Background(), validRequest())

	if !errors.Is(err, svcauth.ErrDuplicateEmail) {
		t.Errorf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestRegisterService_DuplicateUsername(t *testing.T) {
	repo := &fakeRepo{usernameExists: true}
	svc := svcauth.NewRegisterService(repo, &fakeMailer{}, "http://localhost:3000")

	_, err := svc.Register(context.Background(), validRequest())

	if !errors.Is(err, svcauth.ErrDuplicateUsername) {
		t.Errorf("expected ErrDuplicateUsername, got %v", err)
	}
}

func TestRegisterService_RepoCreateError(t *testing.T) {
	repo := &fakeRepo{createErr: errors.New("db down")}
	svc := svcauth.NewRegisterService(repo, &fakeMailer{}, "http://localhost:3000")

	_, err := svc.Register(context.Background(), validRequest())

	if err == nil {
		t.Fatal("expected error from repo.Create, got nil")
	}
}
