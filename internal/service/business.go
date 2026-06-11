package business

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/knowe666/shortener/internal/repository"
)

const maxGenerateAttempts = 10

// URLRepository - интерфейс репозитория
type URLRepository interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, error)
}

// URLShortenerService - сервис бизнес-логики
type URLShortenerService struct {
	repo    URLRepository
	baseURL string
	mu      sync.Mutex
	cache   map[string]string // originalURL -> shortID (для быстрого поиска дубликатов)
}

// NewURLShortenerService создаёт новый сервис
func NewURLShortenerService(repo URLRepository, baseURL string) *URLShortenerService {
	return &URLShortenerService{
		repo:    repo,
		baseURL: baseURL,
		cache:   make(map[string]string),
	}
}

func (s *URLShortenerService) GetOriginalURL(shortID string) (string, error) {
	return s.repo.Get(shortID)
}

var (
	ErrInvalidURL         = errors.New("invalid URL: must start with http:// or https://")
	ErrFailedToGenerateID = errors.New("failed to generate unique short ID (possible ID space exhaustion)")
	ErrDuplicate          = errors.New("url already exists")
	ErrFailedToSave       = errors.New("failed to save URL")
	ErrNotFound           = errors.New("short URL not found")
)

// генерация коротких ссылок - бизнес-логика
func (s *URLShortenerService) CreateShortURL(originalURL string) (string, error) {
	originalURL = strings.TrimSpace(originalURL)
	// Валидация URL
	if originalURL == "" {
		return "", ErrInvalidURL
	}
	// Проверяем, что это HTTP или HTTPS URL
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		return "", ErrInvalidURL
	}
	s.mu.Lock() // для предотвращения race condition
	oldshortID, exists := s.cache[originalURL]
	if exists {
		s.mu.Unlock()
		return url.JoinPath(s.baseURL, oldshortID) // Возвращаем существующую короткую ссылку
	}
	s.mu.Unlock()
	// Генерируем новый ID
	for range maxGenerateAttempts {
		id, err := generateShortID()
		if err != nil {
			continue // попробуем снова, ошибка генерации
		}
		if err := s.repo.Save(id, originalURL); err != nil {
			if errors.Is(err, repository.ErrDuplicateID) {
				continue // ID уже существует, пробуем снова
			}
			return "", fmt.Errorf("%w: %v", ErrFailedToSave, err)
		}
		s.mu.Lock()
		s.cache[originalURL] = id // Сохраняем в кэш
		s.mu.Unlock()
		return url.JoinPath(s.baseURL, id)
	}
	return "", ErrFailedToGenerateID
}

// generateShortID создаёт случайный идентификатор
func generateShortID() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}
