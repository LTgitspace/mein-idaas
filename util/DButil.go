package util

import (
	"fmt"
	"log"
	"time" // <--- Added this for connection lifetime settings

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"mein-idaas/model"
)

func InitDB() *gorm.DB {
	// 1. CONFIGURATION
	host := getEnv("DB_HOST", "localhost")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "xiaomi14T")
	dbName := getEnv("DB_NAME", "idaas")
	port := getEnv("DB_PORT", "5432")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// 2. BOOTSTRAP: CREATE DATABASE IF NOT EXISTS
	maintenanceDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=%s",
		host, user, password, port, sslmode)

	tempDB, err := gorm.Open(postgres.Open(maintenanceDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to Postgres instance: %v", err)
	}

	// Check if database exists
	var exists bool
	checkSQL := fmt.Sprintf("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = '%s')", dbName)
	tempDB.Raw(checkSQL).Scan(&exists)

	if !exists {
		log.Printf("Database '%s' not found. Creating...", dbName)
		if err := tempDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName)).Error; err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		log.Println("Database created successfully.")
	}

	// Close maintenance connection
	sqlDB, _ := tempDB.DB()
	sqlDB.Close()

	// 3. CONNECT TO APP DATABASE
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbName, port, sslmode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to application database: %v", err)
	}

	// 4. AUTO MIGRATE
	log.Println("Running AutoMigrate...")
	err = db.AutoMigrate(
		&model.User{},
		&model.Credential{},
		&model.RefreshToken{},
		&model.Role{},
	)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// 5. CONFIGURE CONNECTION POOL
	// We get the underlying sql.DB object to set pool params
	postgresDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying DB object: %v", err)
	}

	// SetMaxOpenConns: Limit max concurrent queries to prevent DB overload
	postgresDB.SetMaxOpenConns(50)

	// SetMaxIdleConns: Keep these open for fast response (essential for auth)
	postgresDB.SetMaxIdleConns(50)

	// SetConnMaxLifetime: Recycle connections every 30 mins to avoid stale connection errors
	postgresDB.SetConnMaxLifetime(30 * time.Minute)

	log.Println("Database connected, migrated, and pool configured!")
	return db
}

// Helper for env vars
//func getEnv(key, fallback string) string {
//	if value, exists := os.LookupEnv(key); exists {
//		return value
//	}
//	return fallback
//}
