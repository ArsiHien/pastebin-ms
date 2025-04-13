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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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
