package transport

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/knowe666/shortener/internal/repository"
	business "github.com/knowe666/shortener/internal/service"
)

// Используем реальные компоненты вместо моков
func TestURLHandler_HandlePost_Integrated(t *testing.T) {
	// Создаём реальные компоненты
	repo := repository.NewInMemoryURLRepository()
	service := business.NewURLShortenerService(repo, "http://localhost:8080")
	handler := NewURLHandler(service)

	tests := []struct {
		name                 string
		requestBody          string
		expectedStatus       int
		expectedBodyContains string
	}{
		{
			name:                 "Valid HTTP URL",
			requestBody:          "http://example.com",
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "http://localhost:8080/",
		},
		{
			name:                 "Valid HTTPS URL",
			requestBody:          "https://secure.com",
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "http://localhost:8080/",
		},
		{
			name:                 "Empty body",
			requestBody:          "",
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: "Empty URL",
		},
		{
			name:                 "Invalid URL",
			requestBody:          "not-a-valid-url",
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: "Invalid URL",
		},
		{
			name:                 "Whitespace URL",
			requestBody:          "   ",
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: "Empty URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.requestBody))
			w := httptest.NewRecorder()

			handler.HandlePost(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("HandlePost() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			body, err := io.ReadAll(w.Body)
			if err != nil {
				t.Errorf("Failed to read body: %v", err)
			}

			if tt.expectedBodyContains != "" && !bytes.Contains(body, []byte(tt.expectedBodyContains)) {
				t.Errorf("Body = %s, should contain %s", string(body), tt.expectedBodyContains)
			}
		})
	}
}

func TestURLHandler_HandleGet_Integrated(t *testing.T) {
	// Создаём реальные компоненты и сохраняем тестовые данные
	repo := repository.NewInMemoryURLRepository()
	service := business.NewURLShortenerService(repo, "http://localhost:8080")
	handler := NewURLHandler(service)

	// Создаём реальную короткую ссылку через сервис
	shortURL, err := service.CreateShortURL("https://test-redirect.com")
	if err != nil {
		t.Fatalf("Failed to create test URL: %v", err)
	}

	// Извлекаем ID из URL
	shortID := shortURL[len(shortURL)-8:]

	tests := []struct {
		name             string
		shortID          string
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:             "Valid short ID",
			shortID:          shortID,
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://test-redirect.com",
		},
		{
			name:             "Invalid short ID",
			shortID:          "notexist",
			expectedStatus:   http.StatusNotFound,
			expectedLocation: "",
		},
		{
			name:             "Empty ID",
			shortID:          "",
			expectedStatus:   http.StatusBadRequest,
			expectedLocation: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/"+tt.shortID, nil)
			w := httptest.NewRecorder()

			handler.HandleGet(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("HandleGet() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedLocation != "" {
				location := w.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("HandleGet() Location = %v, want %v", location, tt.expectedLocation)
				}
			}
		})
	}
}

// Интеграционный тест полного цикла
func TestIntegration_CompleteFlow(t *testing.T) {
	// Создаём все реальные компоненты
	repo := repository.NewInMemoryURLRepository()
	service := business.NewURLShortenerService(repo, "http://localhost:8080")
	handler := NewURLHandler(service)
	router := SetupRouter(handler)

	// 1. Создаём короткую ссылку
	originalURL := "https://integration-test.com"
	reqBody := bytes.NewBufferString(originalURL)
	req := httptest.NewRequest(http.MethodPost, "/", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create status = %v, want %v", w.Code, http.StatusCreated)
	}

	shortURL := w.Body.String()

	// 2. Извлекаем ID из короткой ссылки
	if len(shortURL) < 8 {
		t.Fatalf("Short URL too short: %s", shortURL)
	}
	shortID := shortURL[len(shortURL)-8:]

	// 3. Переходим по короткой ссылке
	req = httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Redirect status = %v, want %v", w.Code, http.StatusTemporaryRedirect)
	}

	location := w.Header().Get("Location")
	if location != originalURL {
		t.Errorf("Redirect location = %v, want %v", location, originalURL)
	}
}

// Тест нескольких последовательных запросов
func TestMultipleRequests(t *testing.T) {
	repo := repository.NewInMemoryURLRepository()
	service := business.NewURLShortenerService(repo, "http://localhost:8080")
	handler := NewURLHandler(service)

	urls := []string{
		"https://site1.com",
		"https://site2.com",
		"https://site3.com",
	}

	shortIDs := make([]string, len(urls))

	// Создаём несколько ссылок
	for i, url := range urls {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
		w := httptest.NewRecorder()
		handler.HandlePost(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Failed to create URL %s: status %d", url, w.Code)
		}

		shortURL := w.Body.String()
		if len(shortURL) >= 8 {
			shortIDs[i] = shortURL[len(shortURL)-8:]
		}
	}

	// Проверяем, что все созданные ссылки работают
	for i, shortID := range shortIDs {
		req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
		w := httptest.NewRecorder()
		handler.HandleGet(w, req)

		if w.Code != http.StatusTemporaryRedirect {
			t.Errorf("Failed to redirect ID %s: status %d", shortID, w.Code)
		}

		location := w.Header().Get("Location")
		if location != urls[i] {
			t.Errorf("Wrong redirect: got %s, want %s", location, urls[i])
		}
	}
}
