package transport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/knowe666/shortener/internal/repository"
	business "github.com/knowe666/shortener/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock сервиса для тестирования транспортного слоя
type MockURLService struct {
	CreateShortURLFunc func(originalURL string) (string, error)
	GetOriginalURLFunc func(shortID string) (string, error)
}

func (m *MockURLService) CreateShortURL(originalURL string) (string, error) {
	if m.CreateShortURLFunc != nil {
		return m.CreateShortURLFunc(originalURL)
	}
	return "http://localhost:8080/test123", nil
}

func (m *MockURLService) GetOriginalURL(shortID string) (string, error) {
	if m.GetOriginalURLFunc != nil {
		return m.GetOriginalURLFunc(shortID)
	}
	return "https://example.com", nil
}

func TestHandleAPIShorten_Success(t *testing.T) {
	// Создаем мок сервис
	mockService := &MockURLService{
		CreateShortURLFunc: func(originalURL string) (string, error) {
			return "http://localhost:8080/abc123", nil
		},
	}

	handler := NewURLHandler(mockService)

	// Создаем JSON запрос
	reqBody := shortenRequest{
		URL: "https://practicum.yandex.ru",
	}
	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	// Создаем HTTP запрос
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder
	w := httptest.NewRecorder()

	// Вызываем хендлер
	handler.HandleAPIShorten(w, req)

	// Проверяем ответ
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Проверяем JSON ответ
	var response shortenResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/abc123", response.Result)
}

func TestHandleAPIShorten_InvalidJSON(t *testing.T) {
	mockService := &MockURLService{}
	handler := NewURLHandler(mockService)

	// Некорректный JSON
	invalidJSON := []byte(`{"url": "invalid json`)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleAPIShorten(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
}

func TestHandleAPIShorten_WrongContentType(t *testing.T) {
	mockService := &MockURLService{}
	handler := NewURLHandler(mockService)

	reqBody := shortenRequest{URL: "https://example.com"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "text/plain") // Неправильный Content-Type

	w := httptest.NewRecorder()
	handler.HandleAPIShorten(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandleAPIShorten_EmptyURL(t *testing.T) {
	mockService := &MockURLService{}
	handler := NewURLHandler(mockService)

	reqBody := shortenRequest{URL: ""}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleAPIShorten(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandleAPIShorten_DuplicateURL(t *testing.T) {
	mockService := &MockURLService{
		CreateShortURLFunc: func(originalURL string) (string, error) {
			return "", business.ErrDuplicate
		},
	}

	handler := NewURLHandler(mockService)

	reqBody := shortenRequest{URL: "https://example.com"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleAPIShorten(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestHandleAPIShorten_InvalidURL(t *testing.T) {
	mockService := &MockURLService{
		CreateShortURLFunc: func(originalURL string) (string, error) {
			return "", business.ErrInvalidURL
		},
	}

	handler := NewURLHandler(mockService)

	reqBody := shortenRequest{URL: "not-a-valid-url"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleAPIShorten(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Интеграционный тест с реальным роутером
func TestAPIShortenRouteIntegration(t *testing.T) {
	mockService := &MockURLService{
		CreateShortURLFunc: func(originalURL string) (string, error) {
			return "http://localhost:8080/test123", nil
		},
	}

	handler := NewURLHandler(mockService)
	router := SetupRouter(handler)

	// Создаем запрос к новому эндпоинту
	reqBody := shortenRequest{URL: "https://example.com"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var response shortenResponse
	err := json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/test123", response.Result)
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
				m.GetOriginalURLFunc = func(shortID string) (string, error) {
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
				m.GetOriginalURLFunc = func(shortID string) (string, error) {
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
	mockService.GetOriginalURLFunc = func(shortID string) (string, error) {
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
	if service == nil {
		t.Fatal("Service is nil")
	}
	handler := NewURLHandler(service)
	router := SetupRouter(handler)
	// 1. Создаём короткую ссылку
	testURL := "https://integration-test.com"
	reqBody := bytes.NewBufferString(testURL)
	req := httptest.NewRequest(http.MethodPost, "/", reqBody)
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
