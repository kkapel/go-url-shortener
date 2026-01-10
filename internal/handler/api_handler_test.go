package handler

import (
	"fmt"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/repository"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			body:    "https://practicum.yandex.ru/",
		},
	}

	// Создаем репозиторий и конфиг
	repo := repository.NewURLRepository()
	testCfg := &config.Config{
		Host:       "localhost:8080",
		GetURLHost: "http://localhost:8080",
	}
	h := &Handler{
		Repo: repo,
		Cfg:  testCfg,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestPost := httptest.NewRequest(http.MethodPost, test.request, strings.NewReader(test.body))
			postRecorder := httptest.NewRecorder()
			h.APIPagePost(postRecorder, requestPost)

			result := postRecorder.Result()

			assert.Equal(t, test.wantpost.code, result.StatusCode)
			assert.Equal(t, test.wantpost.contentType, result.Header.Get("Content-Type"))

			// Запоминаем короткий URL для Get-запроса
			shortURL := postRecorder.Body.String()

			fmt.Println(shortURL)

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
