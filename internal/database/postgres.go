package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectPostGIS initializes the connection to the PostGIS database
func ConnectPostGIS() *gorm.DB {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// We set TimeZone to Asia/Jakarta to match your operational area
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		host, user, password, dbname, port)

	var db *gorm.DB
	var err error

	// Resilience: Retry loop for Docker/Cloud startup latency
	maxRetries := 5
	for i := 1; i <= maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn), // Keep logs lean in production
		})

		if err == nil {
			break
		}

		log.Printf("⚠️ Database connection failed (Attempt %d/%d). Retrying in 3 seconds...", i, maxRetries)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("❌ Fatal: Could not connect to PostGIS after %d attempts: %v", maxRetries, err)
	}

	// Extract the generic SQL database object to tune the connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ Failed to get database instance: %v", err)
	}

	// Lean Bare-Metal Optimization:
	// Prevent the Go microservice from overwhelming Postgres with too many idle/open sockets
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("✅ Successfully connected to PostGIS Spatial Database")
	return db
}
