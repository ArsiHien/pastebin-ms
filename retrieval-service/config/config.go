package config

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	Port        string
	MongoURI    string
	MongoDBName string
	RedisURI    string
	RabbitMQURI string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
		log.Println("No .env file found, skipping")
	}

	cfg := &Config{
		Port:        getEnv("PORT", "8082"),
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName: getEnv("MONGO_DB_NAME", "pastebin"),
		RedisURI:    getEnv("REDIS_URI", "redis://localhost:6380"),
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
