package router

import (
	"fmt"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/encoding"
	"go-url-shortener/internal/handler"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func Run() error {
	if err := loger.Initialize("INFO"); err != nil {
		return err
	}
	defer loger.Log.Sync()
	loger.Log.Info("Init server start")
	fmt.Println("Init server start")
	cfg := config.CreateConfig()
	repo := repository.CreateRepository()
	urlLocal := repository.NewURLRepository()
	err := repository.InitDB(cfg.DBString)

	if err != nil {
		return err
	}
	// Закрываем БД-соединение
	defer repository.Close()

	loger.Log.Info("Init server running")
	fmt.Println("Init server running")

	h := &handler.Handler{
		Cfg: cfg,
		Rep: repo,
		URL: urlLocal,
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
	r.Use(cookies.RequestCookies)
	r.Post("/", h.APIPagePost)
	r.Post("/api/shorten", h.APIPagePostJSON)
	r.Post("/api/shorten/batch", h.APIPagePostBatch)
	r.Get("/{id}", h.APIPageGet)
	r.Get("/ping", h.APIGetPing)
	r.Get("/api/user/urls", h.APIPageGetUserURLs)
	r.Delete("/api/user/urls", h.APIDeleteURLs)

	return srv.ListenAndServe()
}
