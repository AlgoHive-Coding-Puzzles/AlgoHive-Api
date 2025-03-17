package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
    ApiPort          string
    AllowedOrigins   string
    DefaultPassword  string
    BeeApis          string
    PostgresHost     string
    PostgresPort     string
    PostgresUser     string
    PostgresPassword string
    PostgresDB       string
    RedisHost        string
    RedisPort        string
    RedisPassword    string
    RedisDB          int
    JWTSecret        string
    JWTExpiration    int
)

func LoadConfig() {
    err := godotenv.Load()

    ApiPort = getEnv("API_PORT", "8080")
    AllowedOrigins = getEnv("ALLOWED_ORIGINS", "*")
    DefaultPassword = getEnv("DEFAULT_PASSWORD", "password")
    BeeApis = getEnv("BEE_APIS", "http://localhost:5000")
    PostgresHost = getEnv("POSTGRES_HOST", "localhost")
    PostgresPort = getEnv("POSTGRES_PORT", "5432")
    PostgresUser = getEnv("POSTGRES_USER", "postgres")
    PostgresPassword = getEnv("POSTGRES_PASSWORD", "password")
    PostgresDB = getEnv("POSTGRES_DB", "algohive")
    RedisHost = getEnv("CACHE_HOST", "localhost")
    RedisPort = getEnv("CACHE_PORT", "6379")
    RedisPassword = getEnv("CACHE_PASSWORD", "")
    RedisDB = getEnvAsInt("CACHE_DB", 0)
    JWTSecret = getEnv("JWT_SECRET", "your_secret_key")
    JWTExpiration = getEnvAsInt("JWT_EXPIRATION", 86400)

    // Only log a warning if .env file couldn't be loaded
    if err != nil {
        log.Println("Warning: .env file not found, using environment variables or defaults")
    }
    }

func getEnv(key string, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
    if value, exists := os.LookupEnv(key); exists {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}