package handler

import (
	"context"
	"encoding/json"
	"errors"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/service"
	"io"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

type Handler struct {
	Cfg     *config.Config
	Service *service.ShortenerService
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

// APIPagePost - Post запрос на формирование сокращенного URL
func (h *Handler) APIPagePost(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		loger.Log.Info("api_handler.go", zap.String("Function APIPagePost", "Start"))
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		longURL := string(body)
		shortURL, err := h.Service.GetURL(req.Context(), longURL, "short", "Post")
		url, errorJoinPath := url.JoinPath(h.Cfg.GetURLHost, shortURL)

		if err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIPagePost", err.Error()))

			if errors.Is(err, service.ErrConflict) {
				res.WriteHeader(http.StatusConflict)
				res.Write([]byte(url))
				return
			}
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if errorJoinPath != nil {
			loger.Log.Error("api_handler.go", zap.String("errorJoinPath", errorJoinPath.Error()))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return

		}

		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(url))

	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// APIPageGet выполняет Get запрос.
func (h *Handler) APIPageGet(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:

		loger.Log.Info("APIPageGet", zap.Any("Request Body", req.Body))

		shortURL := req.PathValue("id")

		shortURLDeleted, err := h.Service.CheckFlagDeleteExists(context.Background(), shortURL)

		if err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIPageGet", err.Error()))
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if shortURLDeleted {
			res.WriteHeader(http.StatusGone)
			return
		}

		longURL, err := h.Service.GetURL(req.Context(), shortURL, "long", "Get")

		if err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIPageGet", err.Error()))
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

// APIPagePost - Post запрос на формирование сокращенного URL
// В теле запроса формат JSON
func (h *Handler) APIPagePostJSON(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		var url URL
		var resultJSON ResultJSON
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		loger.Log.Info("APIPagePostJSON", zap.Any("Request Body", body))

		if err = json.Unmarshal(body, &url); err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIPagePostJSON", err.Error()))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		shortURL, err := h.Service.GetURL(req.Context(), url.URL, "short", "Post")

		if err != nil && !errors.Is(err, service.ErrConflict) {
			loger.Log.Error("api_handler.go", zap.String("Function APIPagePostJSON", err.Error()))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		isConflict := errors.Is(err, service.ErrConflict)

		//Фомрмируем ответ
		resultJSON.Result = h.Cfg.GetURLHost + "/" + shortURL
		loger.Log.Info("APIPagePostJSON", zap.String("short_url", shortURL))
		resp, err := json.Marshal(resultJSON)

		if err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIPagePostJSON", err.Error()))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		if isConflict {
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

// APIGetPing - Get-запрос на доступность сервера
func (h *Handler) APIGetPing(res http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodGet:

		if err := h.Service.CheckConnect(req.Context()); err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIGetPing", err.Error()))
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)

	default:
		errorResponse(res)
	}

}

// APIPagePost - Post запрос на формирование сокращенного URL
// В теле запроса формат JSON
// Поддерживает батч-данные
func (h *Handler) APIPagePostBatch(res http.ResponseWriter, req *http.Request) {
	loger.Log.Info("APIPagePostBatch starts")

	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса (JSON-массив)
		var batchJSON []BatchJSON
		var BatchJSONResponseVar []BatchJSONResponse
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		if err = json.Unmarshal(body, &batchJSON); err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIPagePostBatch", err.Error()))

			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		loger.Log.Info("APIPagePostBatch", zap.Any("Request Body", batchJSON))

		//Получаем короткий URL
		//Проходим по циклу оригинальных(длинных) URL
		var shortURL string
		for i := range batchJSON {
			shortURL, err = h.Service.GetURL(req.Context(), batchJSON[i].OriginalURL, "short", "Post")

			if err != nil {
				loger.Log.Error("Ошибка в методе GetURL", zap.String("error", err.Error()))

				http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
			loger.Log.Error("api_handler.go", zap.String("Function APIPagePostBatch", err.Error()))

			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
			if err == http.ErrNoCookie {
				res.WriteHeader(http.StatusNoContent)
				return
			} else {
				http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}

		id, err := cookies.GetUserID(cookie.Value)

		loger.Log.Info("APIPageGetUserURLs", zap.Int("input id", id))

		if err != nil {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Получаем список url-ов из БД
		urls, err := h.Service.GetURLsByUserID(req.Context(), id)

		if err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIPageGetUserURLs", err.Error()))

			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if len(urls) == 0 {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		// Заполняем ответ
		var urlsResponse []ShortURLByUserResponse
		for shortURL, longURL := range urls {
			shortURLByUserResponseVar := &ShortURLByUserResponse{
				ShortURL:    h.Cfg.GetURLHost + "/" + shortURL,
				OriginalURL: longURL,
			}

			urlsResponse = append(urlsResponse, *shortURLByUserResponseVar)
		}

		resp, err := json.Marshal(urlsResponse)

		if err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIPageGetUserURLs", err.Error()))

			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

// APIDeleteURLs - функция удаления коротких URL
func (h *Handler) APIDeleteURLs(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodDelete:
		// Читаем тело запроса (JSON-массив)
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		// Получаем из куки UserID
		// Можно потом вынести в отдельный метод
		cookie, err := req.Cookie("access_token")
		if err != nil {
			if err == http.ErrNoCookie {
				res.WriteHeader(http.StatusNoContent)
				return
			} else {
				http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}

		id, err := cookies.GetUserID(cookie.Value)

		loger.Log.Info("APIPageGetUserURLs", zap.Int("input id", id))

		if err != nil {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var arrayURLs []string
		// Парсим
		if err := json.Unmarshal(body, &arrayURLs); err != nil {
			loger.Log.Error("api_handler.go", zap.String("Function APIDeleteURLs", err.Error()))

			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Взаимодействуем с сервисом и БД в отдельной go-рутине
		go h.Service.DeleteURLs(id, arrayURLs)

		res.WriteHeader(http.StatusAccepted)

	default:
		errorResponse(res)
	}
}

func errorResponse(res http.ResponseWriter) {
	res.WriteHeader(http.StatusBadRequest)
}
