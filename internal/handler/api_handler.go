package handler

import (
	"encoding/json"
	"go-url-shortener/internal/config"
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
	DB  *repository.DB
	URL *repository.URL
}

type URL struct {
	URL string `json:"url"`
}

type ResultJSON struct {
	Result string `json:"result"`
}

func (h *Handler) APIPagePost(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}

		longURL := string(body)
		shortURL, err := service.GetURL(longURL, h.Cfg.FileStoragePath, "short", &h.Rep.Mu, h.Cfg.DBString, h.DB, req, h.URL)

		if err != nil {
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

		shortURL := req.PathValue("id")
		longURL, err := service.GetURL(shortURL, h.Cfg.FileStoragePath, "long", &h.Rep.Mu, h.Cfg.DBString, h.DB, req, h.URL)

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

		if err := json.Unmarshal(body, &url); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		shortURL, err := service.GetURL(url.URL, h.Cfg.FileStoragePath, "short", &h.Rep.Mu, h.Cfg.DBString, h.DB, req, h.URL)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		//Фомрмируем ответ
		resultJSON.Result = h.Cfg.GetURLHost + "/" + shortURL
		loger.Log.Info("APIPagePostJSON", zap.String("short_url", shortURL))
		resp, err := json.Marshal(resultJSON)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		res.WriteHeader(http.StatusCreated)
		loger.Log.Info("APIPagePostJSON", zap.String("result", string(resp)))
		res.Write(resp)

	default:
		errorResponse(res)
	}
}

func (h *Handler) APIGetPing(res http.ResponseWriter, req *http.Request) {

	if h.DB == nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch req.Method {
	case http.MethodGet:

		if err := h.DB.CheckConnect(); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
		}
		res.WriteHeader(http.StatusOK)

	default:
		errorResponse(res)
	}

}

func errorResponse(res http.ResponseWriter) {
	res.WriteHeader(http.StatusBadRequest)
}
