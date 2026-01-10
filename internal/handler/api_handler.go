package handler

import (
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

func (h *Handler) APIPagePost(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		body, err := io.ReadAll(req.Body)
		if err != nil {
			// Дописать обработку ошибки
			return
		}

		longURL := string(body)
		shortURL := h.Repo.GetShortURL(longURL)

		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(h.Cfg.GetURLHost + shortURL))
		fmt.Println(body)

	default:

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
		errorResponse(res, req)
	}

}

func errorResponse(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusBadRequest)
}
