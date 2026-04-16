package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"

	"superset/auth-service/configs"
	svcauth "superset/auth-service/internal/app/auth"
	svcdb "superset/auth-service/internal/app/db"
	delivery "superset/auth-service/internal/delivery/http"
	httpauth "superset/auth-service/internal/delivery/http/auth"
	httpdb "superset/auth-service/internal/delivery/http/db"
	"superset/auth-service/internal/domain/auth"
	domaindb "superset/auth-service/internal/domain/db"
	"superset/auth-service/internal/pkg/email"
	repopostgres "superset/auth-service/internal/repository/postgres"
	reporedis "superset/auth-service/internal/repository/redis"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := configs.Load()
	log.Printf("Loaded config: DB DSN=%s, SMTP Host=%s, App BaseURL=%s, Redis URL=%s",
		cfg.DB.DSN, cfg.SMTP.Host, cfg.App.BaseURL, cfg.Redis.URL)
	if cfg.DB.CredentialsEncryptionKey == "" {
		log.Fatal("DB_CREDENTIALS_ENCRYPTION_KEY must be set")
	}
	// Database
	db, err := gorm.Open(gormpostgres.Open(cfg.DB.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(
		&auth.RegisterUser{},
		&auth.User{},
		&auth.Role{},
		&auth.Permission{},
		&auth.ViewMenu{},
		&auth.PermissionView{},
		&domaindb.Database{},
	); err != nil {
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

	pubBlock, _ := pem.Decode([]byte(cfg.JWT.PublicKeyPEM))
	if pubBlock == nil {
		log.Fatal("failed to parse JWT_PUBLIC_KEY PEM")
	}
	pubKeyAny, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		log.Fatalf("failed to parse RSA public key: %v", err)
	}
	pubKey, ok := pubKeyAny.(*rsa.PublicKey)
	if !ok {
		log.Fatal("JWT_PUBLIC_KEY is not an RSA public key")
	}

	// Wire dependencies
	registerRepo := repopostgres.NewRegisterUserRepository(db)
	verifyRepo := repopostgres.NewVerifyRepository(db)
	loginRepo := repopostgres.NewLoginRepository(db)
	userRepo := repopostgres.NewUserRepository(db)
	userAdminRepo := repopostgres.NewUserAdminRepository(db)
	roleRepo := repopostgres.NewRoleRepository(db)
	rbacPermissionRepo := repopostgres.NewRBACPermissionRepository(db)
	userRoleRepo := repopostgres.NewUserRoleRepository(db)
	permissionRepo := repopostgres.NewPermissionRepository(db)
	databaseRepo := repopostgres.NewDatabaseRepository(db)
	rateRepo := reporedis.NewRateLimitRepository(redisClient)
	jwtRepo := reporedis.NewJWTRepository(redisClient)
	refreshRepo := reporedis.NewRefreshRepository(redisClient)
	roleCacheRepo := reporedis.NewRoleCacheRepository(redisClient)
	rbacPermissionCacheRepo := reporedis.NewRBACPermissionCacheRepository(redisClient)

	mailer := email.NewSMTPSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)

	registerSvc := svcauth.NewRegisterService(registerRepo, mailer, cfg.App.BaseURL)
	verifySvc := svcauth.NewVerifyService(verifyRepo)
	loginSvc := svcauth.NewLoginService(loginRepo, rateRepo, refreshRepo, privKey)
	refreshSvc := svcauth.NewRefreshService(refreshRepo, userRepo, privKey)
	logoutSvc := svcauth.NewLogoutService(jwtRepo, refreshRepo)
	userSvc := svcauth.NewUserService(userAdminRepo, roleCacheRepo)
	roleSvc := svcauth.NewRoleService(roleRepo, roleCacheRepo)
	userRoleSvc := svcauth.NewUserRoleService(userRoleRepo, roleCacheRepo)
	permissionSvc := svcauth.NewPermissionService(permissionRepo, roleCacheRepo)
	databaseSvc, err := svcdb.NewDatabaseService(databaseRepo, nil, nil, cfg.DB.CredentialsEncryptionKey)
	if err != nil {
		log.Fatalf("failed to initialize database service: %v", err)
	}
	if err := permissionSvc.SeedDefaults(context.Background()); err != nil {
		log.Fatalf("failed to seed permission views: %v", err)
	}

	registerHandler := httpauth.NewRegisterHandler(registerSvc)
	verifyHandler := httpauth.NewVerifyHandler(verifySvc, cfg.App.BaseURL)
	loginHandler := httpauth.NewLoginHandler(loginSvc)
	refreshHandler := httpauth.NewRefreshHandler(refreshSvc)
	logoutHandler := httpauth.NewLogoutHandler(logoutSvc, pubKey)
	userHandler := httpauth.NewUserHandler(userSvc)
	roleHandler := httpauth.NewRoleHandler(roleSvc)
	userRoleHandler := httpauth.NewUserRoleHandler(userRoleSvc)
	permissionHandler := httpauth.NewPermissionHandler(permissionSvc)
	databaseHandler := httpdb.NewDatabaseHandler(databaseSvc)

	router := delivery.NewRouter(
		registerHandler,
		verifyHandler,
		loginHandler,
		refreshHandler,
		logoutHandler,
		userHandler,
		roleHandler,
		userRoleHandler,
		permissionHandler,
		databaseHandler,
		pubKey,
		jwtRepo,
		userRepo,
		roleRepo,
		rbacPermissionRepo,
		rbacPermissionCacheRepo,
	)

	log.Printf("Auth Service starting on :%s", cfg.App.Port)
	if err := router.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
