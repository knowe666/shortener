package repository

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
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
	uuidMap  map[string]string // shortID -> uuid
}

func NewFileURLRepository(filePath string) (*FileURLRepository, error) {
	repo := &FileURLRepository{
		filePath: filePath,
		urls:     make(map[string]string),
		uuidMap:  make(map[string]string),
	}

	// Загружаем существующие данные из файла
	if err := repo.loadFromFile(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return repo, nil
}

func (r *FileURLRepository) Save(shortID, originalURL string) error {
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
	return r.saveToFile()
}

func (r *FileURLRepository) Get(shortID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if shortID == "" {
		return "", EmptyIDError
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
			UUID:        string(rune(i + 48)), // просто для примера, лучше использовать реальный UUID
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
		r.uuidMap[record.ShortURL] = record.UUID
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
