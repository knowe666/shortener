package main

import (
	"log"
	"net/http"

	config "github.com/knowe666/shortener/internal/config"
	transport "github.com/knowe666/shortener/internal/handler"
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

	log.Printf("Server starting on %s", cfg.ServerAddress)
	log.Printf("Base URL for short links: %s", cfg.BaseURL)
	log.Printf("File storage path: %s", cfg.FileStoragePath)
	log.Printf("Database DSN: %s", cfg.DatabaseDSN) // Логируем DSN (скрывая пароль в продакшене)

	// Выбираем тип репозитория
	var urlRepo repository.URLRepository

	// Приоритет: PostgreSQL > File > In-memory
	if cfg.DatabaseDSN != "" {
		postgresRepo, err := repository.NewPostgresURLRepository(cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("Failed to initialize PostgreSQL repository: %v", err)
		}
		urlRepo = postgresRepo
		log.Printf("Using PostgreSQL storage")

		// Закрываем соединение при завершении
		defer func() {
			if err := postgresRepo.Close(); err != nil {
				log.Printf("Failed to close database connection: %v", err)
			}
		}()
	} else if cfg.FileStoragePath != "" {
		fileRepo, err := repository.NewFileURLRepository(cfg.FileStoragePath)
		if err != nil {
			log.Fatalf("Failed to initialize file repository: %v", err)
		}
		urlRepo = fileRepo
		log.Printf("Using file storage: %s", cfg.FileStoragePath)
	} else {
		urlRepo = repository.NewInMemoryURLRepository()
		log.Printf("Using in-memory storage")
	}

	urlService := business.NewURLShortenerService(urlRepo, cfg.BaseURL)
	urlHandler := transport.NewURLHandler(urlService)
	router := transport.SetupRouter(urlHandler)

	if err := http.ListenAndServe(cfg.ServerAddress, router); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
