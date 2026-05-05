package repository

import (
	"fmt"
	"sync"
)

// Реализация хранения данных в памяти
type InMemoryURLRepository struct {
	mu    sync.RWMutex
	urls  map[string]string // shortID -> originalURL
	cache map[string]string // originalURL -> shortID
}

func NewInMemoryURLRepository() *InMemoryURLRepository {
	return &InMemoryURLRepository{
		urls:  make(map[string]string),
		cache: make(map[string]string),
	}
}

func (r *InMemoryURLRepository) Save(shortID, originalURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.urls[shortID] = originalURL
	r.cache[originalURL] = shortID
	return nil
}

func (r *InMemoryURLRepository) FindByShortID(shortID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	originalURL, exists := r.urls[shortID]
	if !exists {
		return "", fmt.Errorf("short URL not found")
	}
	return originalURL, nil
}

func (r *InMemoryURLRepository) FindByOriginalURL(originalURL string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	shortID, exists := r.cache[originalURL]
	if !exists {
		return "", fmt.Errorf("original URL not found")
	}
	return shortID, nil
}

func (r *InMemoryURLRepository) Exists(shortID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.urls[shortID]
	return exists
}
