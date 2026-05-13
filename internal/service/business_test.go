package business

import (
	"errors"
	"testing"

	"github.com/knowe666/shortener/internal/repository"
)

// Mock репозитория для тестирования бизнес-логики
type MockURLRepository struct {
	urls      map[string]string
	cache     map[string]string
	saveError error
	findError error
}

func NewMockURLRepository() *MockURLRepository {
	return &MockURLRepository{
		urls:  make(map[string]string),
		cache: make(map[string]string),
	}
}

func (m *MockURLRepository) Save(shortID, originalURL string) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.urls[shortID] = originalURL
	m.cache[originalURL] = shortID
	return nil
}

func (m *MockURLRepository) FindByShortID(shortID string) (string, error) {
	if m.findError != nil {
		return "", m.findError
	}
	url, exists := m.urls[shortID]
	if !exists {
		return "", errors.New("not found")
	}
	return url, nil
}

func (m *MockURLRepository) FindByOriginalURL(originalURL string) (string, error) {
	shortID, exists := m.cache[originalURL]
	if !exists {
		return "", errors.New("not found")
	}
	return shortID, nil
}

func (m *MockURLRepository) Exists(shortID string) (bool, error) {
	if _, ok := m.urls[shortID]; ok {
		return true, repository.ErrFailedExistsID
	}
	return false, nil
}

// Установка ошибки для тестирования
func (m *MockURLRepository) SetSaveError(err error) {
	m.saveError = err
}

func TestURLShortenerService_CreateShortURL(t *testing.T) {
	tests := []struct {
		name         string
		originalURL  string
		baseURL      string
		setupMock    func(*MockURLRepository)
		wantErr      bool
		wantContains string
	}{
		{
			name:         "Valid URL creation",
			originalURL:  "https://example.com",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *MockURLRepository) {},
			wantErr:      false,
			wantContains: "http://localhost:8080/",
		},
		{
			name:         "Invalid URL without scheme",
			originalURL:  "example.com",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *MockURLRepository) {},
			wantErr:      true,
			wantContains: "",
		},
		{
			name:         "Empty URL",
			originalURL:  "",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *MockURLRepository) {},
			wantErr:      true,
			wantContains: "",
		},
		{
			name:        "Duplicate URL - should return existing",
			originalURL: "https://duplicate.com",
			baseURL:     "http://localhost:8080",
			setupMock: func(m *MockURLRepository) {
				m.Save("existing123", "https://duplicate.com")
			},
			wantErr:      false,
			wantContains: "http://localhost:8080/existing123",
		},
		{
			name:         "URL with https scheme",
			originalURL:  "https://secure.com",
			baseURL:      "https://short.com",
			setupMock:    func(m *MockURLRepository) {},
			wantErr:      false,
			wantContains: "https://short.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := NewMockURLRepository()
			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			service := NewURLShortenerService(mockRepo, tt.baseURL)
			got, err := service.CreateShortURL(tt.originalURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateShortURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != "" {
				if len(got) < len(tt.wantContains) {
					t.Errorf("CreateShortURL() = %v, want contains %v", got, tt.wantContains)
				}
			}
		})
	}
}

func TestURLShortenerService_GetOriginalURL(t *testing.T) {
	mockRepo := NewMockURLRepository()
	mockRepo.Save("test123", "https://example.com")

	service := NewURLShortenerService(mockRepo, "http://localhost:8080")

	tests := []struct {
		name    string
		shortID string
		want    string
		wantErr bool
	}{
		{
			name:    "Existing short ID",
			shortID: "test123",
			want:    "https://example.com",
			wantErr: false,
		},
		{
			name:    "Non-existing short ID",
			shortID: "notexist",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.GetOriginalURL(tt.shortID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOriginalURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetOriginalURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Тест генерации уникальных ID
func TestGenerateShortID(t *testing.T) {
	// Проверяем, что ID генерируются
	id1, err := generateShortID()
	if err != nil {
		t.Errorf("generateShortID() failed: %v", err)
	}

	if len(id1) != 8 {
		t.Errorf("Generated ID length = %d, want 8", len(id1))
	}

	// Проверяем уникальность
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, _ := generateShortID()
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}
