package repository

import (
	"errors"
	"fmt"
	"sync"
)

// Реализация хранения данных в памяти
type InMemoryURLRepository struct {
	mu    sync.Mutex
	urls  map[string]string // shortID -> originalURL
	cache map[string]string // originalURL -> shortID
}

var (
	ErrFailedToGenerateID = errors.New("failed to generate ID, id exists")
)

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
	r.mu.Lock()
	defer r.mu.Unlock()

	originalURL, ok := r.urls[shortID]
	if !ok {
		return "", fmt.Errorf("short URL not found")
	}
	return originalURL, nil
}

func (r *InMemoryURLRepository) FindByOriginalURL(originalURL string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	shortID, exists := r.cache[originalURL]
	if !exists {
		return "", fmt.Errorf("original URL not found")
	}
	return shortID, nil
}

func (r *InMemoryURLRepository) Exists(shortID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.urls[shortID]
	if ok {
		return false, nil
	} else {
		return true, ErrFailedToGenerateID
	}
}
