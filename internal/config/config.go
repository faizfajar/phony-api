package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DSN           string
	AdminUser     string
	AdminPassword string
}

var AppConfig *Config

func LoadConfig() {
	// Load .env sekali di sini
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system env")
	}

	// Susun DSN dari variabel lingkungan
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USER", "user"),
		getEnv("DB_PASSWORD", "pass"),
		getEnv("DB_NAME", "phony_db"),
		getEnv("DB_PORT", "5432"),
	)

	AppConfig = &Config{
		Port:          getEnv("PORT", "8080"),
		DSN:           dsn,
		AdminUser:     getEnv("ADMIN_USER", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
