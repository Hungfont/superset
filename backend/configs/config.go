package configs

import (
	"os"
)

type Config struct {
	DB    DBConfig
	SMTP  SMTPConfig
	App   AppConfig
}

type DBConfig struct {
	DSN string
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
			DSN: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/superset?sslmode=disable"),
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
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
