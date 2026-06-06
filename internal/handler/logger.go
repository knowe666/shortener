package transport

import (
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var logger *zap.Logger
var sugar *zap.SugaredLogger

// InitLogger инициализирует глобальный логгер
func InitLogger() error {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
		return err
	}
	sugar = logger.Sugar()
	return nil
}

// GetLogger возвращает глобальный логгер
func GetLogger() *zap.Logger {
	if logger == nil {
		if err := InitLogger(); err != nil {
			log.Printf("Failed to init logger: %v", err)
		}
	}
	return logger
}

// GetSugaredLogger возвращает sugared логгер
func GetSugaredLogger() *zap.SugaredLogger {
	if sugar == nil {
		InitLogger()
	}
	return sugar
} // responseWriterWrapper оборачивает http.ResponseWriter для захвата статуса и размера ответа

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

// LoggingMiddleware логирует HTTP запросы
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
