package transport

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzipMiddleware_Compression(t *testing.T) {
	// Инициализируем логгер для тестов
	InitLogger()

	// Тестовый handler, который возвращает JSON ответ
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"test":"data"}`))
	})

	// Тестовый handler для plain text
	textHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`plain text data`))
	})

	tests := []struct {
		name           string
		acceptEncoding string
		contentType    string
		handler        http.Handler
		shouldCompress bool
	}{
		{
			name:           "Client supports gzip and JSON content",
			acceptEncoding: "gzip",
			contentType:    "application/json",
			handler:        testHandler,
			shouldCompress: true,
		},
		{
			name:           "Client supports gzip and HTML content",
			acceptEncoding: "gzip",
			contentType:    "text/html",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`<html>test</html>`))
			}),
			shouldCompress: true,
		},
		{
			name:           "Client supports gzip but plain text",
			acceptEncoding: "gzip",
			contentType:    "text/plain",
			handler:        textHandler,
			shouldCompress: false,
		},
		{
			name:           "Client does not support gzip",
			acceptEncoding: "",
			contentType:    "application/json",
			handler:        testHandler,
			shouldCompress: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := GzipMiddleware(tt.handler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			// Проверяем заголовок Content-Encoding
			contentEncoding := resp.Header.Get("Content-Encoding")
			hasGzip := contentEncoding == "gzip"

			if hasGzip != tt.shouldCompress {
				t.Errorf("Compression = %v, want %v", hasGzip, tt.shouldCompress)
			}

			// Если ответ должен быть сжат, пробуем распаковать
			if tt.shouldCompress {
				gzReader, err := gzip.NewReader(resp.Body)
				if err != nil {
					t.Fatalf("Failed to create gzip reader: %v", err)
				}
				defer gzReader.Close()

				body, err := io.ReadAll(gzReader)
				if err != nil {
					t.Fatalf("Failed to read decompressed body: %v", err)
				}

				if len(body) == 0 {
					t.Errorf("Decompressed body is empty")
				}
			}
		})
	}
}

func TestGzipMiddleware_Decompression(t *testing.T) {
	InitLogger()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		w.Write(body)
	})

	// Сжимаем тестовые данные
	testData := []byte(`{"test":"compressed data"}`)
	var compressedBuf bytes.Buffer
	gzWriter := gzip.NewWriter(&compressedBuf)
	gzWriter.Write(testData)
	gzWriter.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &compressedBuf)
	req.Header.Set("Content-Encoding", "gzip")

	w := httptest.NewRecorder()
	handler := GzipMiddleware(testHandler)
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != string(testData) {
		t.Errorf("Response body = %v, want %v", string(body), string(testData))
	}
}
