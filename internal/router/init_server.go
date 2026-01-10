package router

import (
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/handler"
	"go-url-shortener/internal/repository"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Run() error {
	repo := repository.NewURLRepository()
	cfg := config.CreateConfig()

	h := &handler.Handler{
		Repo: repo,
	}

	r := chi.NewRouter()
	r.Post("/", h.APIPagePost)
	r.Get("/{id}", h.APIPageGet)

	return http.ListenAndServe(cfg.Host, r)
}
