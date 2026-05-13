package business

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/knowe666/shortener/internal/repository"
)

const maxGenerateAttempts = 10

// Интерфейс для связи со слоем данных
type URLRepository interface {
	Save(shortID, originalURL string) error
	FindByShortID(shortID string) (string, error)
	FindByOriginalURL(originalURL string) (string, error)
	Exists(shortID string) (bool, error)
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

	// Проверяем, не существует ли уже такой URL
	if shortID, err := s.repo.FindByOriginalURL(originalURL); err == nil {
		return fmt.Sprintf("%s/%s", s.baseURL, shortID), nil
	}

	// Генерируем новый ID
	var shortID string
	for range maxGenerateAttempts {
		id, err := generateShortID()
		if err != nil {
			fmt.Errorf("failed to generate ID: %w", err)
			continue // попробуем снова, ошибка генерации
		}
		_, err = s.repo.Exists(id)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrFailedExistsID):
				continue
			default:
				fmt.Errorf("Udefind error generate ID: %w", err)
				continue
			}
		}
		shortID = id
		break

	}

	if shortID == "" {
		return "", ErrFailedToGenerateID
	}

	// Сохраняем через репозиторий
	if err := s.repo.Save(shortID, originalURL); err != nil {
		return "", ErrFailedToSave
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
