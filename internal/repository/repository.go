package repository

import (
	"errors"
	"fmt"
	"maps"
	"sync"
)

// URLRepository определяет минимальный интерфейс для работы с хранилищем
type URLRepository interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, error)
}

// Pingable интерфейс для репозиториев, поддерживающих проверку соединения
type Pingable interface {
	Ping() error
}

var (
	ErrNotFoundID  = errors.New("short URL not found for ID")
	ErrEmptyID     = errors.New("short ID cannot be empty")
	ErrDuplicateID = errors.New("short ID already exists")
)

// ErrDuplicateOriginalURL - ошибка, которая возникает при попытке сохранить уже существующий URL
// и содержит существующий short_id
type ErrDuplicateOriginalURL struct {
	ShortID string
}

func (e *ErrDuplicateOriginalURL) Error() string {
	return fmt.Sprintf("original URL already exists (existing short ID: %s)", e.ShortID)
}

// InMemoryURLRepository реализует URLRepository с хранением в памяти
type InMemoryURLRepository struct {
	mu    sync.Mutex
	urls  map[string]string
	cache map[string]string
}

// NewInMemoryURLRepository создаёт новый экземпляр репозитория
func NewInMemoryURLRepository() *InMemoryURLRepository {
	return &InMemoryURLRepository{
		urls:  make(map[string]string),
		cache: make(map[string]string),
	}
}

// Save сохраняет короткую ссылку
func (r *InMemoryURLRepository) Save(shortID, originalURL string) error {
	if shortID == "" {
		return fmt.Errorf("short ID cannot be empty")
	}
	if originalURL == "" {
		return fmt.Errorf("original URL cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем, существует ли уже такой URL
	for id, url := range r.urls {
		if url == originalURL {
			return &ErrDuplicateOriginalURL{ShortID: id}
		}
	}

	if _, ok := r.urls[shortID]; ok {
		return fmt.Errorf("%w: %s", ErrDuplicateID, shortID)
	}
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

func (r *InMemoryURLRepository) GetAll() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[string]string)
	maps.Copy(result, r.urls)
	return result
}

// Ping для in-memory репозитория всегда возвращает nil (всегда работает)
func (r *InMemoryURLRepository) Ping() error {
	return nil
}
