package router

import (
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/encoding"
	"go-url-shortener/internal/handler"
	"go-url-shortener/internal/loger"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func Run() error {
	cfg := config.CreateConfig()
	if err := loger.Initialize("INFO"); err != nil {
		return err
	}
	defer loger.Log.Sync()

	h := &handler.Handler{
		Cfg: cfg,
	}

	r := chi.NewRouter()

	srv := &http.Server{
		Addr:         cfg.Host,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	r.Use(loger.RequestLogger)
	r.Use(encoding.RequestEncoding)
	r.Post("/", h.APIPagePost)
	r.Post("/api/shorten", h.APIPagePostJSON)
	r.Get("/{id}", h.APIPageGet)

	return srv.ListenAndServe()
}
