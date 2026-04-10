package validator

import (
	"fmt"
	"regexp"
)

var (
	reUpper   = regexp.MustCompile(`[A-Z]`)
	reLower   = regexp.MustCompile(`[a-z]`)
	reDigit   = regexp.MustCompile(`[0-9]`)
	reSpecial = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

// ValidatePasswordComplexity returns an error if the password does not meet
// the required complexity rules: ≥12 chars, uppercase, lowercase, digit, special.
func ValidatePasswordComplexity(password string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}
	if !reUpper.MatchString(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !reLower.MatchString(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !reDigit.MatchString(password) {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !reSpecial.MatchString(password) {
		return fmt.Errorf("password must contain at least one special character")
	}
	return nil
}
