package transport

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	business "github.com/knowe666/shortener/internal/service"

	"go.uber.org/zap"
)

var logger *zap.Logger
var sugar *zap.SugaredLogger

// Инициализация логгера
func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	sugar = logger.Sugar()
}

// Интерфейс для связи со слоем бизнес-логики
type URLService interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(shortID string) (string, error)
}

type URLHandler struct {
	service URLService
}

func NewURLHandler(service URLService) *URLHandler {
	return &URLHandler{service: service}
}

func (h *URLHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	log.Print("body " + string(body))
	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}
	shortURL, err := h.service.CreateShortURL(originalURL)
	if err != nil {
		switch {
		case errors.Is(err, business.ErrInvalidURL):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, business.ErrDuplicate):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, business.ErrFailedToGenerateID):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		case errors.Is(err, business.ErrFailedToSave):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (h *URLHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	shortID := strings.TrimPrefix(r.URL.Path, "/")
	if shortID == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}
	originalURL, err := h.service.GetOriginalURL(shortID)
	if err != nil {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

func SetupRouter(handler *URLHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(LoggingMiddleware)
	r.Use(GzipMiddleware)
	r.Use(middleware.Recoverer)
	r.Post("/", handler.HandlePost)
	r.Get("/{id}", handler.HandleGet)
	return r
}
