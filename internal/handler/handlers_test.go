package transport

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/knowe666/shortener/internal/auth"
	"github.com/knowe666/shortener/internal/repository"
	business "github.com/knowe666/shortener/internal/service"
)

// Mock сервиса для тестирования транспортного слоя
type MockURLService struct {
	createShortURLFunc func(originalURL, userID string) (string, error)
	getOriginalURLFunc func(shortID string) (string, error)
	deleteUserURLsFunc func(userID string, shortIDs []string) error
}

func (m *MockURLService) DeleteUserURLs(userID string, shortIDs []string) error {
	if m.deleteUserURLsFunc != nil {
		return m.deleteUserURLsFunc(userID, shortIDs)
	}
	return nil
}

func (m *MockURLService) CreateShortURL(originalURL, userID string) (string, error) {
	if m.createShortURLFunc != nil {
		return m.createShortURLFunc(originalURL, userID)
	}
	return "http://localhost:8080/test123", nil
}

func (m *MockURLService) GetOriginalURL(shortID string) (string, error) {
	if m.getOriginalURLFunc != nil {
		return m.getOriginalURLFunc(shortID)
	}
	return "https://example.com", nil
}

func (m *MockURLService) GetUserURLs(userID string) ([]repository.URLData, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	return []repository.URLData{
		{
			ShortURL:    "http://localhost:8080/abc123",
			OriginalURL: "https://example.com",
		},
	}, nil
}

func TestURLHandler_HandlePost(t *testing.T) {
	InitLogger()

	tests := []struct {
		name           string
		requestBody    string
		setupMock      func(*MockURLService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid URL creation",
			requestBody:    "https://example.com",
			setupMock:      func(m *MockURLService) {},
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://localhost:8080/test123",
		},
		{
			name:           "Empty body",
			requestBody:    "",
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty URL\n",
		},
		{
			name:        "Duplicate URL",
			requestBody: "https://duplicate.com",
			setupMock: func(m *MockURLService) {
				m.createShortURLFunc = func(originalURL, userID string) (string, error) {
					return "", business.DuplicateError
				}
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   business.DuplicateError.Error() + "\n",
		},
		{
			name:        "Invalid URL",
			requestBody: "invalid-url",
			setupMock: func(m *MockURLService) {
				m.createShortURLFunc = func(originalURL, userID string) (string, error) {
					return "", business.InvalidURLError
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   business.InvalidURLError.Error() + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockURLService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			authenticator, _ := auth.NewAuthenticator("test-secret")
			handler := NewURLHandler(mockService, authenticator)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.requestBody))
			w := httptest.NewRecorder()

			handler.HandlePost(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("HandlePost() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("HandlePost() body = %v, want %v", w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestURLHandler_HandleGet(t *testing.T) {
	InitLogger()

	tests := []struct {
		name             string
		shortID          string
		setupMock        func(*MockURLService)
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:    "Valid short ID",
			shortID: "test123",
			setupMock: func(m *MockURLService) {
				m.getOriginalURLFunc = func(shortID string) (string, error) {
					return "https://example.com", nil
				}
			},
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://example.com",
		},
		{
			name:    "Invalid short ID",
			shortID: "notexist",
			setupMock: func(m *MockURLService) {
				m.getOriginalURLFunc = func(shortID string) (string, error) {
					return "", business.NotFoundError
				}
			},
			expectedStatus:   http.StatusNotFound,
			expectedLocation: "",
		},
		{
			name:    "Deleted short ID",
			shortID: "deleted123",
			setupMock: func(m *MockURLService) {
				m.getOriginalURLFunc = func(shortID string) (string, error) {
					return "", repository.DeletedError
				}
			},
			expectedStatus:   http.StatusGone,
			expectedLocation: "",
		},
		{
			name:             "Empty ID",
			shortID:          "",
			setupMock:        nil,
			expectedStatus:   http.StatusBadRequest,
			expectedLocation: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockURLService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			authenticator, _ := auth.NewAuthenticator("test-secret")
			handler := NewURLHandler(mockService, authenticator)

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

func TestSetupRouter(t *testing.T) {
	InitLogger()

	mockService := &MockURLService{}
	authenticator, _ := auth.NewAuthenticator("test-secret")
	handler := NewURLHandler(mockService, authenticator)
	router := SetupRouter(handler)

	// Тестируем POST маршрут
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://example.com"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("POST / status = %v, want %v", w.Code, http.StatusCreated)
	}

	// Тестируем GET маршрут
	mockService.getOriginalURLFunc = func(shortID string) (string, error) {
		return "https://example.com", nil
	}

	req = httptest.NewRequest(http.MethodGet, "/test123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("GET /{id} status = %v, want %v", w.Code, http.StatusTemporaryRedirect)
	}
}

func TestSetupRouter_WithGzip(t *testing.T) {
	InitLogger()

	mockService := &MockURLService{}
	authenticator, _ := auth.NewAuthenticator("test-secret")
	handler := NewURLHandler(mockService, authenticator)
	router := SetupRouter(handler)

	// Тестируем POST с gzip поддержкой
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://example.com"))
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// POST ответ - text/plain, не должен сжиматься
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Errorf("POST response should not be compressed")
	}

	// Создаём хендлер для JSON ответа
	jsonHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"test":"data"}`))
	})

	jsonRouter := SetupRouter(handler)
	jsonRouter.Get("/json", jsonHandler)

	req = httptest.NewRequest(http.MethodGet, "/json", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w = httptest.NewRecorder()
	jsonRouter.ServeHTTP(w, req)

	// JSON ответ должен сжиматься
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("JSON response should be compressed when client supports gzip")
	}
}

// Интеграционный тест
func TestIntegration_CompleteFlow(t *testing.T) {
	InitLogger()

	// Создаём реальные компоненты
	repo := repository.NewInMemoryURLRepository()
	service := business.NewURLShortenerService(repo, "http://localhost:8080")
	if service == nil {
		t.Fatal("Service is nil")
	}
	authenticator, _ := auth.NewAuthenticator("test-secret")
	handler := NewURLHandler(service, authenticator)
	router := SetupRouter(handler)

	// 1. Создаём короткую ссылку
	testURL := "https://integration-test.com"
	reqBody := bytes.NewBufferString(testURL)
	req := httptest.NewRequest(http.MethodPost, "/", reqBody)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create status = %v, want %v", w.Code, http.StatusCreated)
	}

	shortURL := w.Body.String()

	// 2. Извлекаем ID из короткой ссылки
	shortID := strings.TrimPrefix(shortURL, "http://localhost:8080/")
	if shortID == "" {
		t.Fatal("Failed to extract short ID from URL:", shortURL)
	}

	// 3. Переходим по короткой ссылке
	req = httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Redirect status = %v, want %v", w.Code, http.StatusTemporaryRedirect)
	}

	location := w.Header().Get("Location")
	if location != "https://integration-test.com" {
		t.Errorf("Redirect location = %v, want %v", location, "https://integration-test.com")
	}
}
