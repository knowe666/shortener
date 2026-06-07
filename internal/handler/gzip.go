package transport

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
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
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip data", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			r.Body = io.NopCloser(gzReader)
			r.Header.Del("Content-Encoding")
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r) // если gzip не поддерживается, передаём управление дальше без изменений
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gz.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		next.ServeHTTP(&customResponseWriter{ResponseWriter: w, gz: gz}, r)
	})
}
