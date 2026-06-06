package transport

import (
	"compress/gzip"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

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

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
	return w.Writer.Write(b)
}

// responseWriterWrapper оборачивает http.ResponseWriter для захвата статуса и размера ответа
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	bodySize   int
}

func newResponseWriterWrapper(w http.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		bodySize:       0,
	}
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	w.bodySize += len(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// shouldCompressContentType проверяет, нужно ли сжимать данный Content-Type
func shouldCompressContentType(contentType string) bool {
	// Приводим к нижнему регистру для сравнения
	contentTypeLower := strings.ToLower(contentType)
	return strings.Contains(contentTypeLower, "application/json") ||
		strings.Contains(contentTypeLower, "text/html")
}

// compressibleResponseWriter оборачивает ResponseWriter и применяет сжатие только для нужных Content-Type
type compressibleResponseWriter struct {
	http.ResponseWriter
	gzipWriter     *gzip.Writer
	originalWriter http.ResponseWriter
	compressed     bool
	headersWritten bool
}

func (c *compressibleResponseWriter) WriteHeader(statusCode int) {
	if !c.headersWritten {
		contentType := c.Header().Get("Content-Type")
		if shouldCompressContentType(contentType) {
			c.compressed = true
			c.Header().Set("Content-Encoding", "gzip")
			c.Header().Del("Content-Length")
		}
		c.headersWritten = true
	}
	c.originalWriter.WriteHeader(statusCode)
}

func (c *compressibleResponseWriter) Write(b []byte) (int, error) {
	if !c.headersWritten {
		c.WriteHeader(http.StatusOK)
	}

	if c.compressed {
		return c.gzipWriter.Write(b)
	}
	return c.originalWriter.Write(b)
}

// gzipHandleMiddleware обрабатывает как сжатые запросы, так и сжатые ответы
func gzipHandleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Обработка сжатого запроса (клиент отправил сжатые данные)
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			r.Body = gzReader
		}

		// 2. Обработка сжатого ответа (клиент поддерживает gzip)
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// Создаём обёртку, которая будет решать, сжимать или нет
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				// Если не удалось создать gzip writer, продолжаем без сжатия
				next.ServeHTTP(w, r)
				return
			}
			defer gz.Close()

			// Используем compressibleResponseWriter для умного сжатия
			wrapper := &compressibleResponseWriter{
				ResponseWriter: w,
				gzipWriter:     gz,
				originalWriter: w,
				compressed:     false,
				headersWritten: false,
			}

			next.ServeHTTP(wrapper, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := newResponseWriterWrapper(w)
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)
		uri := r.URL.RequestURI()
		method := r.Method
		statusCode := wrapped.statusCode
		bodySize := wrapped.bodySize
		logger.Info("HTTP Request",
			zap.String("uri", uri),
			zap.String("method", method),
			zap.Duration("duration", duration),
			zap.Int("status_code", statusCode),
			zap.Int("response_size", bodySize),
		)
	})
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
	r.Use(gzipHandleMiddleware)
	r.Use(middleware.Recoverer)
	r.Post("/", handler.HandlePost)
	r.Get("/{id}", handler.HandleGet)
	return r
}
