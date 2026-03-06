package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"
	"go-url-shortener/internal/service"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type Handler struct {
	Cfg *config.Config
	Rep *repository.Repsitory
	URL *repository.URL
}

type URL struct {
	URL string `json:"url"`
}

type ResultJSON struct {
	Result string `json:"result"`
}

type BatchJSON struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchJSONResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type ShortURLByUserResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (h *Handler) APIPagePost(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		fmt.Println("Метод ApiPagePost")
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}

		longURL := string(body)
		shortURL, err := service.GetURL(longURL, h.Cfg.FileStoragePath, "short", &h.Rep.Mu, h.Cfg.DBString, req, h.URL)

		if err != nil {
			var uniqueViolationError *repository.UniqueViolationError

			if errors.As(err, &uniqueViolationError) {
				res.WriteHeader(http.StatusConflict)
				res.Write([]byte(h.Cfg.GetURLHost + "/" + uniqueViolationError.LongURL))
				return
			}
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(h.Cfg.GetURLHost + "/" + shortURL))

	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) APIPageGet(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:

		loger.Log.Info("APIPageGet", zap.Any("Request Body", req.Body))

		shortURL := req.PathValue("id")
		longURL, err := service.GetURL(shortURL, h.Cfg.FileStoragePath, "long", &h.Rep.Mu, h.Cfg.DBString, req, h.URL)

		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		loger.Log.Info("APIPageGet", zap.String("shortURL", shortURL))
		loger.Log.Info("APIPageGet", zap.String("LongURL", longURL))

		res.Header().Set("content-type", "text/plain")
		res.Header().Set("Location", longURL)
		//http code 307
		res.WriteHeader(http.StatusTemporaryRedirect)

	default:
		errorResponse(res)
	}

}

func (h *Handler) APIPagePostJSON(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		var url URL
		var resultJSON ResultJSON
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		loger.Log.Info("APIPagePostJSON", zap.Any("Request Body", body))

		if err := json.Unmarshal(body, &url); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		shortURL, err := service.GetURL(url.URL, h.Cfg.FileStoragePath, "short", &h.Rep.Mu, h.Cfg.DBString, req, h.URL)

		var uniqueViolationError *repository.UniqueViolationError

		if err != nil && !errors.As(err, &uniqueViolationError) {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		//Фомрмируем ответ
		// в т.ч. для UniqueViolationError
		resultJSON.Result = h.Cfg.GetURLHost + "/" + shortURL
		loger.Log.Info("APIPagePostJSON", zap.String("short_url", shortURL))
		resp, err := json.Marshal(resultJSON)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		if uniqueViolationError != nil {
			//UniqueViolationError
			res.WriteHeader(http.StatusConflict)
		} else {
			res.WriteHeader(http.StatusCreated)
		}
		loger.Log.Info("APIPagePostJSON", zap.String("result", string(resp)))
		res.Write(resp)

	default:
		errorResponse(res)
	}
}

func (h *Handler) APIGetPing(res http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodGet:

		if err := repository.CheckConnect(); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)

	default:
		errorResponse(res)
	}

}

func (h *Handler) APIPagePostBatch(res http.ResponseWriter, req *http.Request) {
	loger.Log.Info("APIPagePostBatch starts")

	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса (JSON-массив)
		var batchJSON []BatchJSON
		var BatchJSONResponseVar []BatchJSONResponse
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		if err := json.Unmarshal(body, &batchJSON); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		loger.Log.Info("APIPagePostBatch", zap.Any("Request Body", batchJSON))

		//Получаем короткий URL
		//Проходим по циклу оригинальных(длинных) URL
		for i := range batchJSON {
			shortURL, err := service.GetURL(batchJSON[i].OriginalURL, h.Cfg.FileStoragePath, "short", &h.Rep.Mu, h.Cfg.DBString, req, h.URL)

			if err != nil {
				loger.Log.Error("Ошибка в методе GetURL", zap.String("error", err.Error()))
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}

			newBatchJSONResponse := BatchJSONResponse{
				CorrelationID: batchJSON[i].CorrelationID,
				ShortURL:      h.Cfg.GetURLHost + "/" + shortURL,
			}

			BatchJSONResponseVar = append(BatchJSONResponseVar, newBatchJSONResponse)
		}

		// Формируем ответ
		resp, err := json.Marshal(BatchJSONResponseVar)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		res.WriteHeader(http.StatusCreated)
		loger.Log.Info("APIPagePostBatch", zap.String("result", string(resp)))
		res.Write(resp)

	default:
		errorResponse(res)
	}
}

// Хэндлер получения ссылок юзера
func (h *Handler) APIPageGetUserURLs(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		// Получаем из куки UserID
		cookie, err := req.Cookie("access_token")
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		id, err := cookies.GetUserID(cookie.Value)

		loger.Log.Info("APIPageGetUserURLs", zap.Int("input id", id))

		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		// Получаем список url-ов из БД
		urls, err := repository.GetURLsByUserID(req.Context(), id)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		// Заполняем ответ
		var urlsResponse []ShortURLByUserResponse
		for shortURL, longURL := range urls {
			shortURLByUserResponseVar := &ShortURLByUserResponse{
				ShortURL:    shortURL,
				OriginalURL: longURL,
			}

			urlsResponse = append(urlsResponse, *shortURLByUserResponseVar)
		}

		resp, err := json.Marshal(urlsResponse)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		res.WriteHeader(http.StatusOK)
		loger.Log.Info("APIPageGetUserURLs", zap.String("result", string(resp)))
		res.Write(resp)

	default:
		errorResponse(res)
	}

}

func errorResponse(res http.ResponseWriter) {
	res.WriteHeader(http.StatusBadRequest)
}
