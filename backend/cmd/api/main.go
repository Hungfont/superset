package main

import (
	"log"

	"superset/auth-service/configs"
	svcauth "superset/auth-service/internal/app/auth"
	httpauth "superset/auth-service/internal/delivery/http/auth"
	delivery "superset/auth-service/internal/delivery/http"
	"superset/auth-service/internal/domain/auth"
	"superset/auth-service/internal/pkg/email"
	repopostgres "superset/auth-service/internal/repository/postgres"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := configs.Load()

	// Database
	db, err := gorm.Open(gormpostgres.Open(cfg.DB.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&auth.RegisterUser{}, &auth.User{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	// Wire dependencies
	repo := repopostgres.NewRegisterUserRepository(db)
	mailer := email.NewSMTPSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	svc := svcauth.NewRegisterService(repo, mailer, cfg.App.BaseURL)
	registerHandler := httpauth.NewRegisterHandler(svc)

	router := delivery.NewRouter(registerHandler)

	log.Printf("Auth Service starting on :%s", cfg.App.Port)
	if err := router.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
