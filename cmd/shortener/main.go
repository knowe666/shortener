package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
)

type URLShortener struct {
	mu    sync.RWMutex
	urls  map[string]string // shortID -> originalURL
	cache map[string]string // originalURL -> shortID
}

func NewURLShortener() *URLShortener {
	return &URLShortener{
		urls:  make(map[string]string),
		cache: make(map[string]string),
	}
}

// generateShortID создаёт случайный идентификатор длиной 8 символов
func generateShortID() (string, error) {
	bytes := make([]byte, 6) // 6 байт = 8 символов в base64 (без padding)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
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
	fmt.Println("body " + string(body))
	originalURL := strings.TrimSpace(string(body))
	// if originalURL == "" {
	// 	http.Error(w, "Empty URL", http.StatusBadRequest)
	// 	return
	// }

	// // Проверяем валидность URL
	// if !isValidURL(originalURL) {
	// 	http.Error(w, "Invalid URL: must start with http:// or https://", http.StatusBadRequest)
	// 	return
	// }

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

	s.mu.RLock()
	defer s.mu.RUnlock()

	originalURL, exists := s.urls[path]
	if !exists {
		http.Error(w, "Short URL not found", http.StatusBadRequest)
		return
	}

	// Отправляем редирект
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

func main() {
	shortener := NewURLShortener()
	r := chi.NewRouter()
	r.Post("/", shortener.HandlePost)
	r.Get("/{id}", shortener.HandleGet)
	fmt.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
