package transport

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

type customResponseWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (crw *customResponseWriter) Write(b []byte) (int, error) {
	// Проверяем Content-Type для сжатия
	contentType := crw.Header().Get("Content-Type")
	// Проверяем, нужно ли сжимать (application/json или text/html)
	if strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html") {
		return crw.gz.Write(b)
	}
	// Если тип не подходит, пишем без сжатия
	return crw.ResponseWriter.Write(b)
}

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := GetLogger()

		// Логируем заголовки для отладки
		logger.Info("Gzip middleware",
			zap.String("Content-Encoding", r.Header.Get("Content-Encoding")),
			zap.String("Accept-Encoding", r.Header.Get("Accept-Encoding")))

		// Распаковка тела запроса
		contentEncoding := r.Header.Get("Content-Encoding")
		if contentEncoding == "gzip" {
			logger.Info("Decompressing request body")
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				logger.Error("Failed to create gzip reader", zap.Error(err))
				http.Error(w, "Invalid gzip data", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			r.Body = io.NopCloser(gzReader)
			// Удаляем заголовок, чтобы дальше handler не знал о сжатии
			r.Header.Del("Content-Encoding")
		}

		// Проверяем, поддерживает ли клиент gzip для ответа
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			logger.Info("Compressing response")
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length")

			// Оборачиваем ResponseWriter для сжатия ответа
			next.ServeHTTP(&customResponseWriter{ResponseWriter: w, gz: gz}, r)
			return
		}
		// Если gzip не нужен, просто передаём дальше
		next.ServeHTTP(w, r)
	})
}
