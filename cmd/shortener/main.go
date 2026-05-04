package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	config "github.com/knowe666/shortener"
)

type URLShortener struct {
	mu     sync.Mutex
	urls   map[string]string // shortID -> originalURL
	cache  map[string]string // originalURL -> shortID
	config *config.Config    // добавляем конфиг в структуру
}

func NewURLShortener(cfg *config.Config) *URLShortener {
	return &URLShortener{
		urls:   make(map[string]string),
		cache:  make(map[string]string),
		config: cfg,
	}
}

// generateShortID создаёт случайный идентификатор длиной 8 символов
func generateShortID() (string, error) {
	bytes := make([]byte, 6) // 6 байт = 8 символов в base64 (без padding)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("ID generation error %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}

// isValidURL простая проверка, что URL начинается с http:// или https://
func isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func (s *URLShortener) HandlePost(w http.ResponseWriter, r *http.Request) {
	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	log.Print("body " + string(body))
	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	// Проверяем валидность URL
	if !isValidURL(originalURL) {
		http.Error(w, "Invalid URL: must start with http:// or https://", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, не сокращали ли уже этот URL
	if shortID, exists := s.cache[originalURL]; exists {
		shortURL := fmt.Sprintf("http://localhost:8080/%s", shortID)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortURL))
		return
	}

	// Генерируем новый ID
	var shortID string
	for {
		id, err := generateShortID()
		if err != nil {
			http.Error(w, "Failed to generate ID", http.StatusInternalServerError)
			return
		}
		if _, exists := s.urls[id]; !exists {
			shortID = id
			break
		}
	}

	// Сохраняем в оба хранилища
	s.urls[shortID] = originalURL
	s.cache[originalURL] = shortID

	shortURL := fmt.Sprintf("http://localhost:8080/%s", shortID)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (s *URLShortener) HandleGet(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	originalURL, exists := s.urls[path]
	if !exists {
		http.Error(w, "Short URL not found", http.StatusBadRequest)
		return
	}

	// Отправляем редирект
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

func main() {
	// Инициализируем конфигурацию из флагов
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Errorf("Failed to load config: %w\n", err)
		return
	}

	// Выводим информацию о конфигурации
	log.Print("Server starting on %s\n", cfg.ServerAddress)
	log.Print("Base URL for short links: %s\n", cfg.BaseURL)

	// Создаём экземпляр URLShortener с конфигом
	shortener := NewURLShortener(cfg)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/", shortener.HandlePost)
	r.Get("/{id}", shortener.HandleGet)
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		log.Print("Server failed: %v\n", err)
	}
}
