package transport

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipWriter для сжатия ответов
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
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

// shouldCompressContentType проверяет, нужно ли сжимать данный Content-Type
func shouldCompressContentType(contentType string) bool {
	contentTypeLower := strings.ToLower(contentType)
	return strings.Contains(contentTypeLower, "application/json") ||
		strings.Contains(contentTypeLower, "text/html")
}

// GzipMiddleware обрабатывает как сжатые запросы, так и сжатые ответы
func GzipMiddleware(next http.Handler) http.Handler {
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
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
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
