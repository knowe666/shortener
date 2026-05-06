package transport

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/knowe666/shortener/internal/repository"
	business "github.com/knowe666/shortener/internal/service"
)

// Mock сервиса для тестирования транспортного слоя
type MockURLService struct {
	createShortURLFunc func(originalURL string) (string, error)
	getOriginalURLFunc func(shortID string) (string, error)
}

func (m *MockURLService) CreateShortURL(originalURL string) (string, error) {
	if m.createShortURLFunc != nil {
		return m.createShortURLFunc(originalURL)
	}
	return "http://localhost:8080/test123", nil
}

func (m *MockURLService) GetOriginalURL(shortID string) (string, error) {
	if m.getOriginalURLFunc != nil {
		return m.getOriginalURLFunc(shortID)
	}
	return "https://example.com", nil
}

func TestURLHandler_HandlePost(t *testing.T) {
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
		// {
		// 	name:           "Invalid URL",
		// 	requestBody:    "not-a-valid-url",
		// 	setupMock:      nil,
		// 	expectedStatus: http.StatusBadRequest,
		// 	expectedBody:   "Invalid URL: must start with http:// or https://\n",
		// },
		{
			name:        "Service error",
			requestBody: "https://example.com",
			setupMock: func(m *MockURLService) {
				m.createShortURLFunc = func(originalURL string) (string, error) {
					return "", http.ErrBodyNotAllowed
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid URL: must start with http:// or https://\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockURLService{}
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			handler := NewURLHandler(mockService)

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
					return "", http.ErrNoLocation
				}
			},
			expectedStatus:   http.StatusNotFound,
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

			handler := NewURLHandler(mockService)

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
	mockService := &MockURLService{}
	handler := NewURLHandler(mockService)
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

// Интеграционный тест
func TestIntegration_CompleteFlow(t *testing.T) {
	// Создаём реальные компоненты
	repo := repository.NewInMemoryURLRepository()
	service := business.NewURLShortenerService(repo, "http://localhost:8080")
	handler := NewURLHandler(service)
	router := SetupRouter(handler)

	// 1. Создаём короткую ссылку
	reqBody := bytes.NewBufferString("https://integration-test.com")
	req := httptest.NewRequest(http.MethodPost, "/", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create status = %v, want %v", w.Code, http.StatusCreated)
	}

	shortURL := w.Body.String()

	// 2. Извлекаем ID из короткой ссылки
	shortID := shortURL[len(shortURL)-8:]

	// 3. Переходим по короткой ссылке
	req = httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
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
