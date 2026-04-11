package main

import (
	"crypto/x509"
	"encoding/pem"
	"log"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"superset/auth-service/configs"
	svcauth "superset/auth-service/internal/app/auth"
	delivery "superset/auth-service/internal/delivery/http"
	httpauth "superset/auth-service/internal/delivery/http/auth"
	"superset/auth-service/internal/domain/auth"
	"superset/auth-service/internal/pkg/email"
	repopostgres "superset/auth-service/internal/repository/postgres"
	reporedis "superset/auth-service/internal/repository/redis"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

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

	// Redis
	redisOpts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		log.Fatalf("invalid REDIS_URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	// RSA key pair for JWT RS256
	if cfg.JWT.PrivateKeyPEM == "" || cfg.JWT.PublicKeyPEM == "" {
		log.Fatal("JWT_PRIVATE_KEY and JWT_PUBLIC_KEY must be set")
	}
	privBlock, _ := pem.Decode([]byte(cfg.JWT.PrivateKeyPEM))
	if privBlock == nil {
		log.Fatal("failed to parse JWT_PRIVATE_KEY PEM")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		log.Fatalf("failed to parse RSA private key: %v", err)
	}

	// Wire dependencies
	registerRepo := repopostgres.NewRegisterUserRepository(db)
	verifyRepo := repopostgres.NewVerifyRepository(db)
	loginRepo := repopostgres.NewLoginRepository(db)
	rateRepo := reporedis.NewRateLimitRepository(redisClient)

	mailer := email.NewSMTPSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)

	registerSvc := svcauth.NewRegisterService(registerRepo, mailer, cfg.App.BaseURL)
	verifySvc := svcauth.NewVerifyService(verifyRepo)
	loginSvc := svcauth.NewLoginService(loginRepo, rateRepo, privKey)

	registerHandler := httpauth.NewRegisterHandler(registerSvc)
	verifyHandler := httpauth.NewVerifyHandler(verifySvc, cfg.App.BaseURL)
	loginHandler := httpauth.NewLoginHandler(loginSvc)

	router := delivery.NewRouter(registerHandler, verifyHandler, loginHandler)

	log.Printf("Auth Service starting on :%s", cfg.App.Port)
	if err := router.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
