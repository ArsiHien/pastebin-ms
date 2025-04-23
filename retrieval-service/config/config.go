package config

import (
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
	// Chỉ load .env file nếu đang chạy ở môi trường dev
	if os.Getenv("ENV") == "dev" {
			log.Println("Loading environment variables from .env file")
			if err := godotenv.Load(); err != nil {
					log.Println("Warning: failed to load .env file")
			}
	}

	config := &Config{
			Port:        os.Getenv("PORT"),
			MongoURI:    os.Getenv("MONGO_URI"),
			MongoDBName: os.Getenv("MONGO_DB_NAME"),
			RedisURI:    os.Getenv("REDIS_URI"),
			RabbitMQURI: os.Getenv("RABBITMQ_URI"),
	}

	return config, nil
}
