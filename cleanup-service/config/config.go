package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	MySQLDSN             string
	MongoURI             string
	MongoDBName          string
	RabbitMQURI          string
	AnalyticsMongoURI    string
	AnalyticsMongoDBName string
	RetrieveMongoURI     string
	RetrieveMongoDBName  string
	RetrieveRedisURI     string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	cfg := &Config{
		Port:                 getEnv("PORT", "8083"),
		MySQLDSN:             getEnv("CREATE_MYSQL_DSN", "user:password@tcp(localhost:3306)/pastebin?parseTime=true"),
		MongoURI:             getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName:          getEnv("MONGO_DB_NAME", "pastebin"),
		RabbitMQURI:          getEnv("RABBITMQ_URI", "amqp://guest:guest@localhost:5672/"),
		AnalyticsMongoURI:    getEnv("ANALYTICS_MONGO_URI", "mongodb://localhost:27017"),
		AnalyticsMongoDBName: getEnv("ANALYTICS_MONGO_DB_NAME", "pastebin_analytics"),
		RetrieveMongoURI:     getEnv("RETRIEVE_MONGO_URI", "mongodb://localhost:27017"),
		RetrieveMongoDBName:  getEnv("RETRIEVE_MONGO_DB_NAME", "pastebin_retrieval"),
		RetrieveRedisURI:     getEnv("RETRIEVE_REDIS_URI", "redis://localhost:6379/0"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
