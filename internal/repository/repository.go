package repository

import (
	"errors"
	"fmt"
	"sync"
)

// URLRepository определяет минимальный интерфейс для работы с хранилищем
type URLRepository interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, error)
}

var (
	ErrNotFoundID = errors.New("short URL not found for ID")
	ErrEmptyID    = errors.New("short ID cannot be empty")
)

// InMemoryURLRepository реализует URLRepository с хранением в памяти
type InMemoryURLRepository struct {
	mu   sync.Mutex
	urls map[string]string // shortID -> originalURL
}

// NewInMemoryURLRepository создаёт новый экземпляр репозитория
func NewInMemoryURLRepository() *InMemoryURLRepository {
	return &InMemoryURLRepository{
		urls: make(map[string]string),
	}
}

// Save сохраняет короткую ссылку
func (r *InMemoryURLRepository) Save(shortID, originalURL string) error {
	if shortID == "" {
		fmt.Println("short ID cannot be empty")
		return fmt.Errorf("short ID cannot be empty")
	}
	if originalURL == "" {
		fmt.Println("original URL cannot be empty")
		return fmt.Errorf("original URL cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.urls[shortID] = originalURL
	return nil
}

// Get возвращает оригинальный URL по короткому ID
func (r *InMemoryURLRepository) Get(shortID string) (string, error) {
	if shortID == "" {
		return "", ErrEmptyID
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	originalURL, exists := r.urls[shortID]
	if !exists {
		return "", ErrNotFoundID
	}
	return originalURL, nil
}
