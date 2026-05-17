package business

import (
	"testing"

	"github.com/knowe666/shortener/internal/repository"
)

func TestURLShortenerService_CreateShortURL(t *testing.T) {
	tests := []struct {
		name         string
		originalURL  string
		baseURL      string
		setupMock    func(*repository.InMemoryURLRepository)
		wantErr      bool
		wantContains string
	}{
		{
			name:         "Invalid URL without scheme",
			originalURL:  "example.com",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *repository.InMemoryURLRepository) {},
			wantErr:      true,
			wantContains: "",
		},
		{
			name:         "Empty URL",
			originalURL:  "",
			baseURL:      "http://localhost:8080",
			setupMock:    func(m *repository.InMemoryURLRepository) {},
			wantErr:      true,
			wantContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repository.NewInMemoryURLRepository()
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
