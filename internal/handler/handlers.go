package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/knowe666/shortener/internal/auth"

	business "github.com/knowe666/shortener/internal/service"
	"go.uber.org/zap"
)

// URLHandler обработчик HTTP запросов
type URLHandler struct {
	service URLService
	logger  *zap.Logger
}

func NewURLHandler(service URLService) *URLHandler {
	return &URLHandler{
		service: service,
		logger:  GetLogger(),
	}
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

func (h *URLHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetOrCreateUserID(w, r)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(string(body))
	h.logger.Info("POST request", zap.String("url", originalURL))

	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.CreateShortURL(originalURL, userID)
	if err != nil {
		h.logger.Error("Failed to create short URL", zap.Error(err))
		switch {
		case errors.Is(err, business.InvalidURLError):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, business.DuplicateError):
			// Возвращаем 409 Conflict с существующим коротким URL
			h.logger.Info("Duplicate URL detected, returning existing short URL",
				zap.String("short_url", shortURL))
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			if _, err := w.Write([]byte(shortURL)); err != nil {
				h.logger.Error("Failed to write response", zap.Error(err))
			}
			return
		case errors.Is(err, business.FailedToGenerateIDError):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		case errors.Is(err, business.FailedToSaveError):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	h.logger.Info("Short URL created", zap.String("short_url", shortURL))
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(shortURL)); err != nil {
		h.logger.Error("Failed to write response", zap.Error(err))
	}
}

func (h *URLHandler) HandleAPIShorten(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetOrCreateUserID(w, r)
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var req shortenRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		h.logger.Error("Failed to decode JSON request", zap.Error(err))
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.CreateShortURL(req.URL, userID)
	if err != nil {
		switch {
		case errors.Is(err, business.InvalidURLError):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, business.DuplicateError):
			// Возвращаем 409 Conflict с существующим коротким URL в JSON формате
			h.logger.Info("Duplicate URL detected, returning existing short URL",
				zap.String("short_url", shortURL))
			response := shortenResponse{
				Result: shortURL,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				h.logger.Error("Failed to encode JSON response", zap.Error(err))
			}
			return
		case errors.Is(err, business.FailedToGenerateIDError):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		case errors.Is(err, business.FailedToSaveError):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := shortenResponse{
		Result: shortURL,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (h *URLHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	// Получаем ID из пути
	shortID := strings.TrimPrefix(r.URL.Path, "/")
	h.logger.Info("GET request", zap.String("path", r.URL.Path), zap.String("shortID", shortID))

	if shortID == "" {
		h.logger.Warn("Empty short ID")
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	originalURL, err := h.service.GetOriginalURL(shortID)
	if err != nil {
		h.logger.Warn("Short URL not found", zap.String("shortID", shortID), zap.Error(err))
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}

	h.logger.Info("Redirecting", zap.String("shortID", shortID), zap.String("originalURL", originalURL))
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// HandlePing проверяет соединение с базой данных
func (h *URLHandler) HandlePing(w http.ResponseWriter, r *http.Request) {
	// Проверяем, поддерживает ли репозиторий Ping
	pingable, ok := h.service.(interface{ Ping() error })
	if !ok {
		// Если репозиторий не поддерживает Ping, считаем что всё работает
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			h.logger.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	if err := pingable.Ping(); err != nil {
		h.logger.Error("Database ping failed", zap.Error(err))
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		h.logger.Error("Failed to write response", zap.Error(err))
	}
}
