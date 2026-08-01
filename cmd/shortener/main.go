package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/knowe666/shortener/internal/auth"
	config "github.com/knowe666/shortener/internal/config"
	transport "github.com/knowe666/shortener/internal/handler"
	"github.com/knowe666/shortener/internal/migrate"
	repository "github.com/knowe666/shortener/internal/repository"
	business "github.com/knowe666/shortener/internal/service"
)

func main() {
	if err := transport.InitLogger(); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	// Загрузка конфигурации
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}
	authenticator, err := auth.NewAuthenticator(cfg.AuthSecret)
	if err != nil {
		log.Fatal("Failed to initialize auth: ", err)
	}
	log.Printf("Server starting on %s", cfg.ServerAddress)
	log.Printf("Base URL for short links: %s", cfg.BaseURL)

	// Выбираем тип репозитория
	var urlRepo repository.URLRepository
	var cleanup func() error

	// Приоритет: PostgreSQL > File > In-Memory
	if cfg.HasDatabase() {
		log.Printf("Using PostgreSQL storage")

		// Выполняем миграции
		if err := migrate.RunMigrations(cfg.DatabaseDSN); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		pgRepo, err := repository.NewPostgresURLRepository(cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("Failed to initialize PostgreSQL repository: %v", err)
		}
		urlRepo = pgRepo
		cleanup = pgRepo.Close
		log.Printf("Connected to PostgreSQL")
	} else if cfg.HasFileStorage() {
		log.Printf("Using file storage: %s", cfg.FileStoragePath)
		fileRepo, err := repository.NewFileURLRepository(cfg.FileStoragePath)
		if err != nil {
			log.Fatalf("Failed to initialize file repository: %v", err)
		}
		urlRepo = fileRepo
	} else {
		log.Printf("Using in-memory storage")
		urlRepo = repository.NewInMemoryURLRepository()
	}

	urlService := business.NewURLShortenerService(urlRepo, cfg.BaseURL)
	urlHandler := transport.NewURLHandler(urlService, authenticator)
	router := transport.SetupRouter(urlHandler)

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server is running on %s", cfg.ServerAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Ждем сигнал остановки
	<-stop
	log.Println("Shutting down server...")

	// Закрываем соединение с БД если есть
	if cleanup != nil {
		if err := cleanup(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}

	log.Println("Server stopped")
}
