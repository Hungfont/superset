package configs

import (
	"os"
)

type Config struct {
	DB    DBConfig
	SMTP  SMTPConfig
	App   AppConfig
	Redis RedisConfig
	JWT   JWTConfig
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	PrivateKeyPEM string
	PublicKeyPEM  string
}

type DBConfig struct {
	DSN                      string
	CredentialsEncryptionKey string
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type AppConfig struct {
	BaseURL string
	Port    string
}

func Load() Config {
	return Config{
		DB: DBConfig{
			DSN:                      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/superset?sslmode=disable"),
			CredentialsEncryptionKey: getEnv("DB_CREDENTIALS_ENCRYPTION_KEY", ""),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "localhost"),
			Port:     getEnv("SMTP_PORT", "1025"),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@superset.local"),
		},
		App: AppConfig{
			BaseURL: getEnv("APP_BASE_URL", "http://localhost:3000"),
			Port:    getEnv("APP_PORT", "8080"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
		},
		JWT: JWTConfig{
			PrivateKeyPEM: getEnv("JWT_PRIVATE_KEY", ""),
			PublicKeyPEM:  getEnv("JWT_PUBLIC_KEY", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
