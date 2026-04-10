package auth

import "context"

// RegisterUserRepository handles persistence of pending registrations.
type RegisterUserRepository interface {
	// EmailExists returns true if the email is taken in ab_user or ab_register_user.
	EmailExists(ctx context.Context, email string) (bool, error)
	// UsernameExists returns true if the username is taken in ab_user or ab_register_user.
	UsernameExists(ctx context.Context, username string) (bool, error)
	// Create persists a new pending registration.
	Create(ctx context.Context, r *RegisterUser) error
}
