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

	urlRepo := repository.NewInMemoryURLRepository()
	urlService := business.NewURLShortenerService(urlRepo, cfg.BaseURL)
	urlHandler := transport.NewURLHandler(urlService)
	router := transport.SetupRouter(urlHandler)

	if err := http.ListenAndServe(cfg.ServerAddress, router); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
