package router

import (
	"go-url-shortener/internal/handler"
	"go-url-shortener/internal/repository"
	"net/http"
)

func Run() error {
	repo := repository.NewURLRepository()

	h := &handler.Handler{
		Repo: repo,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(`/`, h.APIPagePost)
	mux.HandleFunc(`/{id}`, h.APIPageGet)

	return http.ListenAndServe(`:8080`, mux)

}
