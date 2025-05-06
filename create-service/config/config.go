package config

import (
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/eventbus"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/repository"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/worker"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
	"strconv"
)

type AppConfig struct {
	MySQLDSN    string
	RabbitMQURI string
	Port        string
	URLLength   int
}

type App struct {
	Config               *AppConfig
	DB                   *gorm.DB
	RabbitConn           *amqp.Connection
	PasteRepo            paste.Repository
	ExpirationPolicyRepo paste.ExpirationPolicyRepository
	Publisher            paste.EventPublisher
	MySQLSaveWorker      *worker.MySQLSaveWorker // Thêm worker
}

func LoadConfig() *AppConfig {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, using environment variables")
	}

	return &AppConfig{
		MySQLDSN: getEnv("MYSQL_DSN",
			"root:123456@tcp(localhost:3307)/pastebin?charset=utf8mb4"+
				"&parseTime=True"),
		RabbitMQURI: getEnv("RABBITMQ_URI", "amqp://guest:guest@localhost:5672/"),
		Port:        getEnv("PORT", "8080"),
		URLLength:   getIntEnv("URL_LENGTH", 5),
	}
}

func Initialize(cfg *AppConfig) (*App, error) {
	app := &App{Config: cfg}

	if err := setupDatabase(app); err != nil {
		return nil, err
	}

	if err := setupRabbitMQ(app); err != nil {
		Cleanup(app)
		return nil, err
	}

	setupRepositories(app)

	if err := setupWorker(app); err != nil {
		Cleanup(app)
		return nil, err
	}

	return app, nil
}

func setupDatabase(app *App) error {
	db, err := gorm.Open(mysql.Open(app.Config.MySQLDSN), &gorm.Config{})
	if err != nil {
		return err
	}
	app.DB = db

	// Run migrations
	if err := db.AutoMigrate(&paste.ExpirationPolicy{}, &paste.Paste{}); err != nil {
		return err
	}
	log.Println("Database migration completed successfully!")

	return nil
}

func setupRabbitMQ(app *App) error {
	conn, err := amqp.Dial(app.Config.RabbitMQURI)
	if err != nil {
		return err
	}
	app.RabbitConn = conn

	publisher, err := eventbus.NewRabbitMQPublisher(conn)
	if err != nil {
		return err
	}
	app.Publisher = publisher

	return nil
}

func setupRepositories(app *App) {
	app.ExpirationPolicyRepo = repository.NewExpirationPolicyMySQLRepository(app.DB)
	app.PasteRepo = repository.NewPasteMySQLRepository(app.DB)
}

func setupWorker(app *App) error {
	saveWorker, err := worker.NewMySQLSaveWorker(app.RabbitConn, app.PasteRepo)
	if err != nil {
		log.Printf("Failed to create MySQL save saveWorker: %v", err)
		return err
	}
	app.MySQLSaveWorker = saveWorker

	if err := saveWorker.Start(); err != nil {
		log.Printf("Failed to start MySQL save saveWorker: %v", err)
		return err
	}
	log.Println("MySQL save saveWorker started successfully!")
	return nil
}

func Cleanup(app *App) {
	if app.MySQLSaveWorker != nil {
		if err := app.MySQLSaveWorker.Stop(); err != nil {
			log.Printf("Error stopping MySQL save worker: %v", err)
		} else {
			log.Println("MySQL save worker stopped")
		}
	}

	if app.Publisher != nil {
		if err := app.Publisher.Close(); err != nil {
			log.Printf("Error closing publisher: %v", err)
		}
	}

	if app.RabbitConn != nil {
		if err := app.RabbitConn.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		} else {
			log.Println("RabbitMQ disconnected")
		}
	}

	if app.DB != nil {
		sqlDB, err := app.DB.DB()
		if err != nil {
			log.Printf("Error getting SQL DB: %v", err)
		} else {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Error closing database: %v", err)
			} else {
				log.Println("Database disconnected")
			}
		}
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return fallback
}
