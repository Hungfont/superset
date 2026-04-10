package validator_test

import (
	"testing"

	"superset/auth-service/internal/pkg/validator"
)

func TestValidatePasswordComplexity(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "StrongP@ss1!",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Short1!A",
			wantErr:  true,
		},
		{
			name:     "missing uppercase",
			password: "weakpassword1!",
			wantErr:  true,
		},
		{
			name:     "missing lowercase",
			password: "WEAKPASSWORD1!",
			wantErr:  true,
		},
		{
			name:     "missing digit",
			password: "WeakPassword!!",
			wantErr:  true,
		},
		{
			name:     "missing special char",
			password: "WeakPassword12",
			wantErr:  true,
		},
		{
			name:     "exactly 12 chars valid",
			password: "Abcdefg1234!",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePasswordComplexity(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordComplexity(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}
