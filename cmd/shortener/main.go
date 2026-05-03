package main

import (
<<<<<<< HEAD
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	endpoint := "http://localhost:8080/"
	// контейнер данных для запроса
	data := url.Values{}
	// приглашение в консоли
	fmt.Println("Введите длинный URL")
	// открываем потоковое чтение из консоли
	reader := bufio.NewReader(os.Stdin)
	// читаем строку из консоли
	long, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	long = strings.TrimSuffix(long, "\n")
	// заполняем контейнер данными
	data.Set("url", long)
	// добавляем HTTP-клиент
	client := &http.Client{}
	// пишем запрос
	// запрос методом POST должен, помимо заголовков, содержать тело
	// тело должно быть источником потокового чтения io.Reader
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	// в заголовках запроса указываем кодировку
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// отправляем запрос и получаем ответ
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	// выводим код ответа
	fmt.Println("Статус-код ", response.Status)
	defer response.Body.Close()
	// читаем поток из тела ответа
	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	// и печатаем его
	fmt.Println(string(body))
=======
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type URLShortener struct {
	mu    sync.RWMutex
	urls  map[string]string // shortID -> originalURL
	cache map[string]string // originalURL -> shortID
}

func NewURLShortener() *URLShortener {
	return &URLShortener{
		urls:  make(map[string]string),
		cache: make(map[string]string),
	}
}

// generateShortID создаёт случайный идентификатор длиной 8 символов
func generateShortID() (string, error) {
	bytes := make([]byte, 6) // 6 байт = 8 символов в base64 (без padding)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}

// isValidURL простая проверка, что URL начинается с http:// или https://
func isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func (s *URLShortener) HandlePost(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request")
	// Проверяем метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	fmt.Println("post method")
	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	fmt.Println("body " + string(body))
	originalURL := strings.TrimSpace(string(body))
	// if originalURL == "" {
	// 	http.Error(w, "Empty URL", http.StatusBadRequest)
	// 	return
	// }

	// // Проверяем валидность URL
	// if !isValidURL(originalURL) {
	// 	http.Error(w, "Invalid URL: must start with http:// or https://", http.StatusBadRequest)
	// 	return
	// }

	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, не сокращали ли уже этот URL
	if shortID, exists := s.cache[originalURL]; exists {
		shortURL := fmt.Sprintf("http://localhost:8080/%s", shortID)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortURL))
		return
	}

	// Генерируем новый ID
	var shortID string
	for {
		id, err := generateShortID()
		if err != nil {
			http.Error(w, "Failed to generate ID", http.StatusInternalServerError)
			return
		}
		if _, exists := s.urls[id]; !exists {
			shortID = id
			break
		}
	}

	// Сохраняем в оба хранилища
	s.urls[shortID] = originalURL
	s.cache[originalURL] = shortID

	shortURL := fmt.Sprintf("http://localhost:8080/%s", shortID)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (s *URLShortener) HandleGet(w http.ResponseWriter, r *http.Request) {
	fmt.Println("get method")
	// Проверяем метод
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	originalURL, exists := s.urls[path]
	if !exists {
		http.Error(w, "Short URL not found", http.StatusBadRequest)
		return
	}

	// Отправляем редирект
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	shortener := NewURLShortener()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", shortener.HandlePost)
	mux.HandleFunc("GET /{id}", shortener.HandleGet)
	fmt.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
>>>>>>> 6b0f57f3e02634b91f9b3c0edbe500df42586340
}
