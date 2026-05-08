package business

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

const maxGenerateAttempts = 10

// Интерфейс для связи со слоем данных
type URLRepository interface {
	Save(shortID, originalURL string) error
	FindByShortID(shortID string) (string, error)
	FindByOriginalURL(originalURL string) (string, error)
	Exists(shortID string) bool
}

type URLShortenerService struct {
	repo    URLRepository
	baseURL string
}

func NewURLShortenerService(repo URLRepository, baseURL string) *URLShortenerService {
	return &URLShortenerService{
		repo:    repo,
		baseURL: baseURL,
	}
}

// генерация коротких ссылок - бизнес-логика
func (s *URLShortenerService) CreateShortURL(originalURL string) (string, error) {
	// Валидация URL
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		return "", fmt.Errorf("invalid URL: must start with http:// or https://")
	}

	// Проверяем, не существует ли уже такой URL
	if shortID, err := s.repo.FindByOriginalURL(originalURL); err == nil {
		return fmt.Sprintf("%s/%s", s.baseURL, shortID), nil
	}

	// Генерируем новый ID
	var shortID string
	for range maxGenerateAttempts {
		id, err := generateShortID()
		if err != nil {
			return "", fmt.Errorf("failed to generate ID: %w", err)
		}
		if !s.repo.Exists(id) {
			shortID = id
			break
		}

		// опционально: логгируем коллизию для мониторинга
		// log.Printf("collision on attempt %d for id: %s", attempt+1, id)
	}

	if shortID == "" {
		return "", fmt.Errorf("failed to generate unique short ID after %d attempts (possible ID space exhaustion)", maxGenerateAttempts)
	}

	// Сохраняем через репозиторий
	if err := s.repo.Save(shortID, originalURL); err != nil {
		return "", fmt.Errorf("failed to save URL: %w", err)
	}

	return url.JoinPath(s.baseURL, shortID)
}

func (s *URLShortenerService) GetOriginalURL(shortID string) (string, error) {
	return s.repo.FindByShortID(shortID)
}

// generateShortID создаёт случайный идентификатор
func generateShortID() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}
