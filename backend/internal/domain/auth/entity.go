package auth

import "time"

// RegisterUser maps to ab_register_user — pending email verification.
type RegisterUser struct {
	ID               uint      `gorm:"primaryKey;autoIncrement"`
	FirstName        string    `gorm:"column:first_name;not null"`
	LastName         string    `gorm:"column:last_name;not null"`
	Username         string    `gorm:"column:username;uniqueIndex;not null"`
	Email            string    `gorm:"column:email;uniqueIndex;not null"`
	Password         string    `gorm:"column:password;not null"` // bcrypt hash
	RegistrationHash string    `gorm:"column:registration_hash;uniqueIndex;not null"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (RegisterUser) TableName() string { return "ab_register_user" }

// User maps to ab_user — activated accounts.
type User struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	FirstName   string    `gorm:"column:first_name;not null"`
	LastName    string    `gorm:"column:last_name;not null"`
	Username    string    `gorm:"column:username;uniqueIndex;not null"`
	Email       string    `gorm:"column:email;uniqueIndex;not null"`
	Password    string    `gorm:"column:password;not null"`
	Active      bool      `gorm:"column:active;default:true"`
	LoginCount  int       `gorm:"column:login_count;default:0"`
	LastLogin   *time.Time `gorm:"column:last_login"`
	CreatedOn   time.Time `gorm:"column:created_on;autoCreateTime"`
	ChangedOn   time.Time `gorm:"column:changed_on;autoUpdateTime"`
}

func (User) TableName() string { return "ab_user" }

// RegisterRequest holds the raw input from the HTTP request.
type RegisterRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name"  binding:"required"`
	Username  string `json:"username"   binding:"required"`
	Email     string `json:"email"      binding:"required,email"`
	Password  string `json:"password"   binding:"required"`
}

// LoginRequest holds credentials from the login HTTP request.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// UserContext is injected into Gin context by the JWT middleware.
type UserContext struct {
	ID       uint
	Username string
	Email    string
	Active   bool
}
