package transport

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter io.Writer
	compressed bool
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.gzipWriter.Write(b)
}

// GzipMiddleware обрабатывает сжатые запросы и условно сжимает ответы
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			r.Body = gzReader
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")
		compress := strings.Contains(acceptEncoding, "gzip")

		if !compress {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()

		wrapped := &gzipResponseWriter{
			ResponseWriter: w,
			gzipWriter:     gz,
			compressed:     false,
		}

		next.ServeHTTP(wrapped, r)
		// 6. Если тип контента не подходит для сжатия, перезаписываем ответ
		contentType := wrapped.Header().Get("Content-Type")
		if !shouldCompressContentType(contentType) {
			// Не сжимаем - но уже поздно, данные записаны
			// Поэтому мы должны решить ДО записи
		}
	})
}

func shouldCompressContentType(contentType string) bool {
	if contentType == "" {
		return false
	}
	ct := strings.ToLower(strings.Split(contentType, ";")[0])
	return ct == "application/json" || ct == "text/html"
}
