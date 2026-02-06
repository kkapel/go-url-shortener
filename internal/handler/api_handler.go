package handler

import (
	"encoding/json"
	"fmt"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/repository"
	"io"
	"net/http"
)

type Handler struct {
	Repo *repository.URL
	Cfg  *config.Config
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
		shortURL := h.Repo.GetShortURL(longURL)

		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(h.Cfg.GetURLHost + "/" + shortURL))
		fmt.Println(body)

	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) APIPageGet(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:

		shortURL := req.PathValue("id")
		longURL := h.Repo.GetLongURL(shortURL)

		fmt.Println("shortURL: " + shortURL)
		fmt.Println("LongURL: " + longURL)

		res.Header().Set("content-type", "text/plain")
		res.Header().Set("Location", longURL)
		//http code 307
		res.WriteHeader(http.StatusTemporaryRedirect)

	default:
		errorResponse(res)
	}

}

func (h *Handler) APIPagePostJson(res http.ResponseWriter, req *http.Request) {
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

		if err := json.Unmarshal(body, &url); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		shortURL := h.Repo.GetShortURL(url.URL)

		//Фомрмируем ответ
		resultJSON.Result = shortURL
		resp, err := json.Marshal(resultJSON)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		res.WriteHeader(http.StatusCreated)
		res.Write(resp)

	default:
		errorResponse(res)
	}
}

func errorResponse(res http.ResponseWriter) {
	res.WriteHeader(http.StatusBadRequest)
}
