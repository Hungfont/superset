package main

import (
	"log"

	"superset/auth-service/configs"
	"superset/auth-service/internal/domain/auth"
	domaindataset "superset/auth-service/internal/domain/dataset"
	domaindb "superset/auth-service/internal/domain/db"
	domainquery "superset/auth-service/internal/domain/query"

	"github.com/joho/godotenv"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := configs.Load()

	db, err := gorm.Open(gormpostgres.New(gormpostgres.Config{
		DSN:                  cfg.DB.DSN,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(
		&auth.RegisterUser{},
		&auth.User{},
		&auth.Role{},
		&auth.Permission{},
		&auth.ViewMenu{},
		&auth.PermissionView{},
		&auth.RLSFilter{},
		&auth.RLSFilterRoleJunction{},
		&auth.RLSFilterTableJunction{},
		&auth.RLSAuditLog{},
		&domaindb.Database{},
		&domaindataset.Dataset{},
		&domaindataset.Column{},
		&domaindataset.SqlMetric{},
		&domainquery.Query{},
		&domainquery.SavedQuery{},
		&domainquery.TabState{},
		&domainquery.TableSchema{},
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_query_sql_gin ON query USING gin (sql gin_trgm_ops)")

	log.Println("Migration completed successfully")
}
