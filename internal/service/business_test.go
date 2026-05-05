package business

import (
	"testing"

	"github.com/knowe666/shortener/internal/repository"
)

func TestURLShortenerService_CreateShortURL(t *testing.T) {
	// Используем реальный репозиторий
	repo := repository.NewInMemoryURLRepository()
	service := NewURLShortenerService(repo, "http://localhost:8080")

	tests := []struct {
		name        string
		originalURL string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Valid HTTP URL",
			originalURL: "http://example.com",
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "Valid HTTPS URL",
			originalURL: "https://secure.com",
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "Invalid URL without scheme",
			originalURL: "example.com",
			wantErr:     true,
			errContains: "must start with http:// or https://",
		},
		{
			name:        "Empty URL",
			originalURL: "",
			wantErr:     true,
			errContains: "must start with http:// or https://",
		},
		{
			name:        "URL with only scheme",
			originalURL: "http://",
			wantErr:     false,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.CreateShortURL(tt.originalURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateShortURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, should contain %v", err, tt.errContains)
				}
			}

			if !tt.wantErr {
				if got == "" {
					t.Error("CreateShortURL() returned empty string")
				}

				// Проверяем, что ID был сохранён
				expectedPrefix := "http://localhost:8080/"
				if len(got) <= len(expectedPrefix) {
					t.Errorf("Short URL too short: %s", got)
				}
			}
		})
	}
}

func TestURLShortenerService_DuplicateURL(t *testing.T) {
	repo := repository.NewInMemoryURLRepository()
	service := NewURLShortenerService(repo, "http://localhost:8080")

	originalURL := "https://duplicate-test.com"

	// Создаём первую ссылку
	first, err := service.CreateShortURL(originalURL)
	if err != nil {
		t.Fatalf("First creation failed: %v", err)
	}

	// Создаём вторую ссылку с тем же URL
	second, err := service.CreateShortURL(originalURL)
	if err != nil {
		t.Fatalf("Second creation failed: %v", err)
	}

	// Проверяем, что вернулась та же ссылка
	if first != second {
		t.Errorf("Duplicate URL returned different short URLs: %s vs %s", first, second)
	}
}

func TestURLShortenerService_GetOriginalURL(t *testing.T) {
	repo := repository.NewInMemoryURLRepository()
	service := NewURLShortenerService(repo, "http://localhost:8080")

	// Создаём несколько ссылок
	urls := map[string]string{
		"abc123": "https://example1.com",
		"def456": "https://example2.com",
		"ghi789": "https://example3.com",
	}

	for shortID, original := range urls {
		err := repo.Save(shortID, original)
		if err != nil {
			t.Fatalf("Failed to save test data: %v", err)
		}
	}

	tests := []struct {
		name    string
		shortID string
		want    string
		wantErr bool
	}{
		{
			name:    "Existing short ID",
			shortID: "abc123",
			want:    "https://example1.com",
			wantErr: false,
		},
		{
			name:    "Another existing ID",
			shortID: "def456",
			want:    "https://example2.com",
			wantErr: false,
		},
		{
			name:    "Non-existing short ID",
			shortID: "notexist",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Empty short ID",
			shortID: "",
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
		id, err := generateShortID()
		if err != nil {
			t.Errorf("Generation failed at iteration %d: %v", i, err)
			continue
		}
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

// Вспомогательная функция
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
