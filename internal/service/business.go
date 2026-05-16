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

// GetOriginalURL implements [transport.URLService].
func (s *URLShortenerService) GetOriginalURL(shortID string) (string, error) {
	panic("unimplemented")
}

// NewURLShortenerService создаёт новый сервис
func NewURLShortenerService(repo URLRepository, baseURL string) *URLShortenerService {
	return &URLShortenerService{
		repo:    repo,
		baseURL: baseURL,
		cache:   make(map[string]string),
	}
}

var (
	ErrInvalidURL         = errors.New("invalid URL: must start with http:// or https://")
	ErrFailedToGenerateID = errors.New("failed to generate unique short ID (possible ID space exhaustion)")
	ErrDuplicate          = errors.New("url already exists")
	ErrFailedToSave       = errors.New("failed to save URL")
)

// генерация коротких ссылок - бизнес-логика
func (s *URLShortenerService) CreateShortURL(originalURL string) (string, error) {
	// Валидация URL
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		return "", fmt.Errorf("invalid URL: must start with http:// or https://")
	}

	// Генерируем новый ID
	var shortID string
	for range maxGenerateAttempts {
		id, err := generateShortID()
		if err != nil {
			continue // попробуем снова, ошибка генерации
		}
		_, err = s.repo.Get(id)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrEmptyID):
				continue
			case errors.Is(err, repository.ErrNotFoundID):
				shortID = id
			default:
				continue
			}
		}
		if shortID != "" {
			break
		}
	}

	if shortID == "" {
		return "", ErrFailedToGenerateID
	}

	// Сохраняем через репозиторий
	if err := s.repo.Save(shortID, originalURL); err != nil {
		return "", fmt.Errorf("ERR: %w", err)
	}

	return url.JoinPath(s.baseURL, shortID)
}

// generateShortID создаёт случайный идентификатор
func generateShortID() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}
