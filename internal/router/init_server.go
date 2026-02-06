package router

import (
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/handler"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Run() error {
	repo := repository.NewURLRepository()
	cfg := config.CreateConfig()
	if err := loger.Initialize("INFO"); err != nil {
		return err
	}
	defer loger.Log.Sync()

	h := &handler.Handler{
		Repo: repo,
		Cfg:  cfg,
	}

	r := chi.NewRouter()
	r.Use(loger.RequestLogger)
	r.Post("/", h.APIPagePost)
	r.Post("/api/shorten", h.APIPagePostJSON)
	r.Get("/{id}", h.APIPageGet)

	return http.ListenAndServe(cfg.Host, r)
}
