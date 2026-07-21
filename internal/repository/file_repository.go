package repository

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/google/uuid"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileURLRepository struct {
	mu       sync.Mutex
	filePath string
	urls     map[string]string // shortID -> originalURL
	cache    map[string]string // originalURL -> shortID
	userURLs map[string][]string
	deleted  map[string]bool // shortID -> флаг удаления
}

func NewFileURLRepository(filePath string) (*FileURLRepository, error) {
	repo := &FileURLRepository{
		filePath: filePath,
		urls:     make(map[string]string),
		cache:    make(map[string]string),
		userURLs: make(map[string][]string),
		deleted:  make(map[string]bool), // Инициализируем карту удалённых
	}

	if err := repo.loadFromFile(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return repo, nil
}

func (r *FileURLRepository) GetUserURLs(userID string) ([]URLData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

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

func (r *FileURLRepository) Save(shortID, originalURL, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if shortID == "" {
		return EmptyIDError
	}
	if originalURL == "" {
		return errors.New("original URL cannot be empty")
	}
	if _, exists := r.urls[shortID]; exists {
		return DuplicateIDError
	}

	r.urls[shortID] = originalURL
	if _, exists := r.cache[originalURL]; !exists {
		r.cache[originalURL] = shortID
	}
	r.userURLs[userID] = append(r.userURLs[userID], shortID)
	return r.saveToFile()
}

func (r *FileURLRepository) Get(shortID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if shortID == "" {
		return "", EmptyIDError
	}

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

func (r *FileURLRepository) saveToFile() error {
	records := make([]URLRecord, 0, len(r.urls))
	i := 1
	for shortID, originalURL := range r.urls {
		records = append(records, URLRecord{
			UUID:        uuid.New().String(),
			ShortURL:    shortID,
			OriginalURL: originalURL,
		})
		i++
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0644)
}

func (r *FileURLRepository) loadFromFile() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	var records []URLRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	for _, record := range records {
		r.urls[record.ShortURL] = record.OriginalURL
		r.cache[record.OriginalURL] = record.ShortURL
	}
	return nil
}

// GetAll возвращает все записи (для тестирования)
func (r *FileURLRepository) GetAll() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[string]string)
	for k, v := range r.urls {
		result[k] = v
	}
	return result
}

func (r *FileURLRepository) DeleteUserURLs(userID string, shortIDs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if userID == "" {
		return errors.New("user ID cannot be empty")
	}
	if len(shortIDs) == 0 {
		return nil
	}

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

	return r.saveToFile()
}
