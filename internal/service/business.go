package business

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/knowe666/shortener/internal/repository"
)

const maxGenerateAttempts = 10

// URLRepository - интерфейс репозитория
type URLRepository interface {
	Save(shortID, originalURL, userID string) error
	Get(shortID string) (string, error)
	GetUserURLs(userID string) ([]repository.URLData, error)
	DeleteUserURLs(userID string, shortIDs []string) error
}

// ExtendedURLRepository - расширенный интерфейс для проверки дубликатов
type ExtendedURLRepository interface {
	URLRepository
	GetByOriginalURL(originalURL string) (string, error)
}

// URLShortenerService - сервис бизнес-логики
type URLShortenerService struct {
	repo    URLRepository
	baseURL string
	mu      sync.Mutex
	cache   map[string]string // originalURL -> shortID (для быстрого поиска дубликатов)
}

type deleteBatchRequest struct {
	userID   string
	shortIDs []string
}

// DeleteBatcher аккумулирует запросы на удаление и отправляет их пачками.
type DeleteBatcher struct {
	DeleteFunc func(userID string, shortIDs []string) error

	enqueueCh     chan deleteBatchRequest
	flushInterval time.Duration
	maxBatchSize  int
	stopCh        chan struct{}
	stopOnce      sync.Once
	wg            sync.WaitGroup

	mu      sync.Mutex
	pending map[string]map[string]struct{}
}

// NewDeleteBatcher создаёт батчер для асинхронного удаления URL.
func NewDeleteBatcher(service *URLShortenerService, maxBatchSize int, flushInterval time.Duration) *DeleteBatcher {
	if maxBatchSize <= 0 {
		maxBatchSize = 100
	}
	if flushInterval <= 0 {
		flushInterval = 200 * time.Millisecond
	}

	return &DeleteBatcher{
		DeleteFunc:    service.DeleteUserURLs,
		enqueueCh:     make(chan deleteBatchRequest, 1024),
		flushInterval: flushInterval,
		maxBatchSize:  maxBatchSize,
		stopCh:        make(chan struct{}),
		pending:       make(map[string]map[string]struct{}),
	}
}

func (b *DeleteBatcher) Start() {
	b.wg.Add(1)
	go b.run()
}

func (b *DeleteBatcher) Stop() {
	b.stopOnce.Do(func() {
		close(b.stopCh)
	})
	b.wg.Wait()
}

func (b *DeleteBatcher) Enqueue(userID string, shortIDs []string) error {
	if userID == "" || len(shortIDs) == 0 {
		return nil
	}

	select {
	case <-b.stopCh:
		return errors.New("delete batcher stopped")
	case b.enqueueCh <- deleteBatchRequest{userID: userID, shortIDs: shortIDs}:
		return nil
	}
}

func (b *DeleteBatcher) run() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopCh:
			b.drainQueue()
			b.flushPending()
			return
		case req := <-b.enqueueCh:
			b.collect(req)
		case <-ticker.C:
			b.flushPending()
		}
	}
}

func (b *DeleteBatcher) drainQueue() {
	for {
		select {
		case req := <-b.enqueueCh:
			b.collect(req)
		default:
			return
		}
	}
}

func (b *DeleteBatcher) collect(req deleteBatchRequest) {
	b.mu.Lock()
	if b.pending[req.userID] == nil {
		b.pending[req.userID] = make(map[string]struct{})
	}
	for _, shortID := range req.shortIDs {
		b.pending[req.userID][shortID] = struct{}{}
	}

	total := 0
	for _, ids := range b.pending {
		total += len(ids)
	}
	if total >= b.maxBatchSize {
		b.mu.Unlock()
		b.flushPending()
		return
	}
	b.mu.Unlock()
}

func (b *DeleteBatcher) flushPending() {
	b.mu.Lock()
	if len(b.pending) == 0 {
		b.mu.Unlock()
		return
	}

	snapshot := make(map[string][]string, len(b.pending))
	for userID, ids := range b.pending {
		batch := make([]string, 0, len(ids))
		for shortID := range ids {
			batch = append(batch, shortID)
		}
		snapshot[userID] = batch
		delete(b.pending, userID)
	}
	b.mu.Unlock()

	for userID, shortIDs := range snapshot {
		if b.DeleteFunc != nil {
			_ = b.DeleteFunc(userID, shortIDs)
		}
	}
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
	originalURL, err := s.repo.Get(shortID)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (s *URLShortenerService) DeleteUserURLs(userID string, shortIDs []string) error {
	if userID == "" {
		return errors.New("user ID cannot be empty")
	}
	if len(shortIDs) == 0 {
		return nil
	}
	return s.repo.DeleteUserURLs(userID, shortIDs)
}

var (
	InvalidURLError         = errors.New("invalid URL: must start with http:// or https://")
	FailedToGenerateIDError = errors.New("failed to generate unique short ID (possible ID space exhaustion)")
	DuplicateError          = errors.New("url already exists")
	FailedToSaveError       = errors.New("failed to save URL")
	NotFoundError           = errors.New("short URL not found")
)

// buildShortURL создает полный короткий URL из baseURL и shortID
func (s *URLShortenerService) buildShortURL(shortID string) string {
	// Простое конкатенирование, так как baseURL всегда валидный
	if strings.HasSuffix(s.baseURL, "/") {
		return s.baseURL + shortID
	}
	return s.baseURL + "/" + shortID
}

// генерация коротких ссылок - бизнес-логика
func (s *URLShortenerService) CreateShortURL(originalURL, userID string) (string, error) {
	originalURL = strings.TrimSpace(originalURL)
	// Валидация URL
	if originalURL == "" {
		return "", InvalidURLError
	}
	// Проверяем, что это HTTP или HTTPS URL
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		return "", InvalidURLError
	}
	s.mu.Lock() // для предотвращения race condition
	oldShortID, exists := s.cache[originalURL]
	if exists {
		s.mu.Unlock()
		fullURL, err := url.JoinPath(s.baseURL, oldShortID)
		if err != nil {
			return "", err
		}
		return fullURL, DuplicateError
	}
	s.mu.Unlock()

	// Проверяем в репозитории (если поддерживает)
	if extRepo, ok := s.repo.(ExtendedURLRepository); ok {
		if shortID, err := extRepo.GetByOriginalURL(originalURL); err == nil {
			s.mu.Lock()
			s.cache[originalURL] = shortID
			s.mu.Unlock()
			fullURL, err := url.JoinPath(s.baseURL, shortID)
			if err != nil {
				return "", err
			}
			return fullURL, DuplicateError
		}
	}

	// Генерируем новый ID
	for range maxGenerateAttempts {
		id, err := generateShortID()
		if err != nil {
			continue // попробуем снова, ошибка генерации
		}
		if err := s.repo.Save(id, originalURL, userID); err != nil {
			// Проверяем, является ли ошибка дубликатом оригинального URL
			var dupErr *repository.ErrDuplicateOriginalURL
			if errors.As(err, &dupErr) {
				// URL уже существует - возвращаем существующий короткий URL
				s.mu.Lock()
				defer s.mu.Unlock()
				s.cache[originalURL] = dupErr.ShortID
				return s.buildShortURL(dupErr.ShortID), DuplicateError
			}

			if errors.Is(err, repository.DuplicateIDError) {
				continue // ID уже существует, пробуем снова
			}
			return "", fmt.Errorf("%w: %v", FailedToSaveError, err)
		}
		s.mu.Lock()
		s.cache[originalURL] = id // Сохраняем в кэш
		s.mu.Unlock()
		fullURL, err := url.JoinPath(s.baseURL, id)
		if err != nil {
			return "", err
		}
		return fullURL, nil
	}
	return "", FailedToGenerateIDError
}

// generateShortID создаёт случайный идентификатор
func generateShortID() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}

func (s *URLShortenerService) GetUserURLs(userID string) ([]repository.URLData, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	urls, err := s.repo.GetUserURLs(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user URLs: %w", err)
	}

	result := make([]repository.URLData, len(urls))
	for i := range urls {
		shortURL := urls[i].ShortURL
		if fullURL, err := url.JoinPath(s.baseURL, shortURL); err == nil {
			shortURL = fullURL
		}
		result[i] = repository.URLData{
			ShortURL:    shortURL,
			OriginalURL: urls[i].OriginalURL,
		}
	}

	return result, nil
}
