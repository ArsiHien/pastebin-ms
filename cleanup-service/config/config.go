package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	Port        string
	MySQLDSN    string
	MongoURI    string
	MongoDBName string
	RabbitMQURI string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	cfg := &Config{
		Port:        getEnv("PORT", "8083"),
		MySQLDSN:    getEnv("MYSQL_DSN", "user:password@tcp(localhost:3306)/pastebin?parseTime=true"),
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName: getEnv("MONGO_DB_NAME", "pastebin"),
		RabbitMQURI: getEnv("RABBITMQ_URI", "amqp://guest:guest@localhost:5672/"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
