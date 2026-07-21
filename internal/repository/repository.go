package repository

import (
	"errors"
	"fmt"
	"maps"
	"sync"
)

// URLRepository определяет минимальный интерфейс для работы с хранилищем
type URLRepository interface {
	Save(shortID, originalURL, userID string) error
	Get(shortID string) (string, error)
	GetUserURLs(userID string) ([]URLData, error)
	DeleteUserURLs(userID string, shortIDs []string) error
}

// Pingable интерфейс для репозиториев, поддерживающих проверку соединения
type Pingable interface {
	Ping() error
}

// URLData представляет данные URL для ответа API
type URLData struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	DeletedFlag bool   `json:"-"`
}

var (
	NotFoundIDError  = errors.New("short URL not found for ID")
	EmptyIDError     = errors.New("short ID cannot be empty")
	DuplicateIDError = errors.New("short ID already exists")
	DeletedError     = errors.New("URL has been deleted")
)

// DuplicateOriginalURLError- ошибка, которая возникает при попытке сохранить уже существующий URL
// и содержит существующий short_id
type ErrDuplicateOriginalURL struct {
	ShortID string
}

func (e *ErrDuplicateOriginalURL) Error() string {
	return fmt.Sprintf("original URL already exists (existing short ID: %s)", e.ShortID)
}

// InMemoryURLRepository реализует URLRepository с хранением в памяти
type InMemoryURLRepository struct {
	mu       sync.Mutex
	urls     map[string]string // shortID -> originalURL
	cache    map[string]string // originalURL -> shortID (обратный мап)
	userURLs map[string][]string
	deleted  map[string]bool // shortID -> флаг удаления
}

func NewInMemoryURLRepository() *InMemoryURLRepository {
	return &InMemoryURLRepository{
		urls:     make(map[string]string),
		cache:    make(map[string]string),
		userURLs: make(map[string][]string),
		deleted:  make(map[string]bool),
	}
}

func (r *InMemoryURLRepository) GetUserURLs(userID string) ([]URLData, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	shortIDs, exists := r.userURLs[userID]
	if !exists || len(shortIDs) == 0 {
		return []URLData{}, nil
	}

	result := make([]URLData, 0, len(shortIDs))
	for _, shortID := range shortIDs {
		// Пропускаем удалённые URL
		if r.deleted[shortID] {
			continue
		}
		originalURL, ok := r.urls[shortID]
		if ok {
			result = append(result, URLData{
				ShortURL:    shortID,
				OriginalURL: originalURL,
			})
		}
	}
	return result, nil
}

func (r *InMemoryURLRepository) DeleteUserURLs(userID string, shortIDs []string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if len(shortIDs) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем владение и помечаем как удалённые
	for _, shortID := range shortIDs {
		found := false
		for _, id := range r.userURLs[userID] {
			if id == shortID {
				found = true
				break
			}
		}
		if found {
			r.deleted[shortID] = true
		}
	}

	return nil
}

// Save сохраняет короткую ссылку
func (r *InMemoryURLRepository) Save(shortID, originalURL, userID string) error {
	if shortID == "" {
		return fmt.Errorf("short ID cannot be empty")
	}
	if originalURL == "" {
		return fmt.Errorf("original URL cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// O(1) проверка дубликата через обратный мап
	if existingID, ok := r.cache[originalURL]; ok {
		return &ErrDuplicateOriginalURL{ShortID: existingID}
	}
	// O(1) проверка на занятый shortID
	if _, ok := r.urls[shortID]; ok {
		return fmt.Errorf("%w: %s", DuplicateIDError, shortID)
	}
	// Сохраняем в оба мапа
	r.urls[shortID] = originalURL
	r.cache[originalURL] = shortID
	r.userURLs[userID] = append(r.userURLs[userID], shortID)
	return nil
}

// Get возвращает оригинальный URL по короткому ID
func (r *InMemoryURLRepository) Get(shortID string) (string, error) {
	if shortID == "" {
		return "", EmptyIDError
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем, не удалён ли URL
	if r.deleted[shortID] {
		return "", DeletedError
	}

	originalURL, exists := r.urls[shortID]
	if !exists {
		return "", NotFoundIDError
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
