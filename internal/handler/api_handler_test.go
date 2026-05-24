package handler

import (
	"encoding/json"
	"fmt"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"
	"go-url-shortener/internal/service"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAPIHandler(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name     string
		wantpost want
		wantget  want
		request  string
		body     string
	}{
		{
			name: "positive test",
			wantpost: want{
				code:        201,
				contentType: "text/plain",
			},
			wantget: want{
				code:        307,
				contentType: "text/plain",
			},
			request: "http://localhost:8080",
			body:    "https://practicum.yandex/",
		},
	}

	// Создаем конфиг
	testCfg := &config.Config{
		Host:            "localhost:8080",
		GetURLHost:      "http://localhost:8080",
		FileStoragePath: filepath.Join(".", "storage.txt"),
	}
	fileRepo := repository.CreateRepository()
	urlLocal := repository.NewURLRepository()
	svc := service.NewShortenerService(nil, fileRepo, urlLocal, testCfg)
	h := &Handler{
		Cfg:     testCfg,
		Service: svc,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			loger.Log.Info("Начинается тест")
			requestPost := httptest.NewRequest(http.MethodPost, test.request, strings.NewReader(test.body))
			postRecorder := httptest.NewRecorder()
			h.APIPagePost(postRecorder, requestPost)

			result := postRecorder.Result()

			assert.Equal(t, test.wantpost.code, result.StatusCode)
			assert.Equal(t, test.wantpost.contentType, result.Header.Get("Content-Type"))

			// Запоминаем короткий URL для Get-запроса
			shortURL := postRecorder.Body.String()

			loger.Log.Info("доп.информация для теста",
				zap.String("shortURL", shortURL))

			// Теперь выполним Get запрос
			requestGet := httptest.NewRequest(http.MethodGet, shortURL, nil)
			requestGet.SetPathValue("id", path.Base(shortURL))
			getRecorder := httptest.NewRecorder()
			h.APIPageGet(getRecorder, requestGet)

			resultGet := getRecorder.Result()

			assert.Equal(t, test.wantget.code, resultGet.StatusCode)
			assert.Equal(t, test.wantget.contentType, resultGet.Header.Get("Content-Type"))

			// Сравним longURL
			assert.Equal(t, test.body, resultGet.Header.Get("Location"))

			errPost := result.Body.Close()
			require.NoError(t, errPost)

			errGet := resultGet.Body.Close()
			require.NoError(t, errGet)

		})
	}
}

func TestAPIHandlerJSON(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name       string
		wantpost   want
		wantget    want
		request    string
		body       string
		url        URL
		bodyResult string
	}{
		{
			name: "positive test",
			wantpost: want{
				code:        201,
				contentType: "application/json",
			},
			wantget: want{
				code:        307,
				contentType: "text/plain",
			},
			request:    "http://localhost:8080",
			body:       `{"url": "https://practicum.yandex/"}`,
			bodyResult: "https://practicum.yandex/",
		},
	}

	// Создаем конфиг
	testCfg := &config.Config{
		Host:            "localhost:8080",
		GetURLHost:      "http://localhost:8080",
		FileStoragePath: filepath.Join(".", "storage.txt"),
	}

	fileRepo := repository.CreateRepository()
	urlLocal := repository.NewURLRepository()
	svc := service.NewShortenerService(nil, fileRepo, urlLocal, testCfg)
	h := &Handler{
		Cfg:     testCfg,
		Service: svc,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var response ResultJSON

			requestPost := httptest.NewRequest(http.MethodPost, test.request, strings.NewReader(test.body))
			postRecorder := httptest.NewRecorder()
			h.APIPagePostJSON(postRecorder, requestPost)

			result := postRecorder.Result()
			loger.Log.Info("result APIPagePostJSON", zap.Any("result APIPagePostJSON", result.Body))

			assert.Equal(t, test.wantpost.code, result.StatusCode)
			assert.Equal(t, test.wantpost.contentType, result.Header.Get("Content-Type"))

			// Запоминаем короткий URL для Get-запроса
			err := json.Unmarshal(postRecorder.Body.Bytes(), &response)
			require.NoError(t, err)

			shortURL := response.Result
			loger.Log.Info("test info", zap.String("short_url", shortURL))

			// Теперь выполним Get запрос
			requestGet := httptest.NewRequest(http.MethodGet, shortURL, nil)
			requestGet.SetPathValue("id", path.Base(shortURL))
			getRecorder := httptest.NewRecorder()
			h.APIPageGet(getRecorder, requestGet)

			resultGet := getRecorder.Result()

			assert.Equal(t, test.wantget.code, resultGet.StatusCode)
			assert.Equal(t, test.wantget.contentType, resultGet.Header.Get("Content-Type"))

			// Сравним longURL
			assert.Equal(t, test.bodyResult, resultGet.Header.Get("Location"))

			errPost := result.Body.Close()
			require.NoError(t, errPost)

			errGet := resultGet.Body.Close()
			require.NoError(t, errGet)

		})
	}
}

func SkipTestAPIPagePostBatch(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name       string
		wantpost   want
		wantget    want
		request    string
		body       string
		url        URL
		bodyResult string
	}{
		{
			name: "positive test",
			wantpost: want{
				code:        201,
				contentType: "application/json",
			},
			wantget: want{
				code:        307,
				contentType: "text/plain",
			},
			request:    "http://localhost:8080",
			body:       `[{"correlation_id":"corel_aaaa","original_url":"https://practicum.yandex/"},{"correlation_id":"corel_bbbb","original_url":"https://google.com"}]`,
			bodyResult: "https://practicum.yandex/",
		},
	}

	// Создаем конфиг
	testCfg := &config.Config{
		Host:            "localhost:8080",
		GetURLHost:      "http://localhost:8080",
		FileStoragePath: filepath.Join(".", "storage.txt"),
		DBString:        "postgres://postgres:admin@localhost:5432/postgres?sslmode=disable",
	}

	defer func() { _ = loger.Log.Sync() }()

	fileRepo := repository.CreateRepository()
	urlLocal := repository.NewURLRepository()
	svc := service.NewShortenerService(nil, fileRepo, urlLocal, testCfg)

	h := &Handler{
		Cfg:     testCfg,
		Service: svc,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestPost := httptest.NewRequest(http.MethodPost, test.request, strings.NewReader(test.body))
			postRecorder := httptest.NewRecorder()
			h.APIPagePostBatch(postRecorder, requestPost)

			result := postRecorder.Result()
			defer func() { _ = result.Body.Close() }()

			assert.Equal(t, test.wantpost.code, result.StatusCode)
			assert.Equal(t, test.wantpost.contentType, result.Header.Get("Content-Type"))

			//
		})
	}

}

func ExampleHandler_APIPagePost() {

	cfg := &config.Config{
		Host:            "localhost",
		GetURLHost:      "http://localhost:8080",
		FileStoragePath: filepath.Join(".", "storage.txt"),
		DBString:        "",
	}
	fileRepo := repository.CreateRepository()
	urlLocal := repository.NewURLRepository()
	databaseInstance, err := repository.InitDB(cfg.DBString)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
	srv := service.NewShortenerService(databaseInstance, fileRepo, urlLocal, cfg)
	// Создаем хэндлер
	h := &Handler{
		Cfg:     cfg,
		Service: srv,
	}

	resp := httptest.NewRecorder()
	requestPost := httptest.NewRequest(http.MethodPost, cfg.GetURLHost, strings.NewReader("https://practicum.yandex/"))

	// Делаем запрос
	h.APIPagePost(resp, requestPost)

	// Выводим результат
	fmt.Printf("Status: %d\n", resp.Code)
	fmt.Printf("Content: %s\n", resp.Header().Get("Content-Type"))

	// Output:
	// Status: 201
	// Content: text/plain

}
