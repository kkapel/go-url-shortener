package router

import (
	"context"
	"go-url-shortener/internal/audit"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/encoding"
	"go-url-shortener/internal/handler"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"
	"go-url-shortener/internal/service"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func Run() error {
	if err := loger.Initialize("INFO"); err != nil {
		return err
	}
	defer func() {
		_ = loger.Log.Sync()
	}()

	loger.Log.Info("Init server start")
	cfg := config.CreateConfig()
	fileRepo := repository.CreateRepository()
	urlLocal := repository.NewURLRepository()
	databaseInstance, err := repository.InitDB(cfg.DBString)
	if err != nil {
		return err
	}

	service := service.NewShortenerService(databaseInstance, fileRepo, urlLocal, cfg)
	cookie := cookies.NewCookie(service)

	// Закрываем БД-соединение
	if databaseInstance != nil {
		defer func() { _ = databaseInstance.Close() }()
	}

	loger.Log.Info("Init server running")

	h := &handler.Handler{
		Cfg:     cfg,
		Service: service,
	}

	r := chi.NewRouter()

	srv := &http.Server{
		Addr:         cfg.Host,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Канал для graceful shutdown
	quit := make(chan os.Signal, 1)
	// Отлавливаем сигналы прерывания (Ctrl+C) и завершения процесса
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	r.Use(loger.RequestLogger)
	r.Use(encoding.RequestEncoding)
	r.Use(cookie.RequestCookies)

	r.Post("/api/shorten/batch", h.APIPagePostBatch)
	r.Get("/ping", h.APIGetPing)
	r.Get("/api/user/urls", h.APIPageGetUserURLs)
	r.Delete("/api/user/urls", h.APIDeleteURLs)

	// Создаем отдельный канал для аудита и запускаем go-рутину
	auditChan := make(chan audit.AuditFormat, 10)
	auditMU := new(sync.Mutex)

	r.Group(func(r chi.Router) {
		r.Use(audit.Audit(auditChan))
		r.Post("/", h.APIPagePost)
		r.Post("/api/shorten", h.APIPagePostJSON)
		r.Get("/{id}", h.APIPageGet)
	})

	go audit.ProcessAudit(auditChan, cfg.FlagAuditFile, auditMU, cfg.FlagAuditURL)

	errCh := make(chan error, 1)
	// Запускаем сервер в отдельной горутине
	go func() {
		if cfg.EnableHttps {
			loger.Log.Info("HTTPS enabled")
			if err := srv.ListenAndServeTLS("cert.pem", "key.pem"); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}
	}()

	loger.Log.Info("Server started successfully")

	// Логика для graceful shutdown
	// Ожидаем сигнал для graceful shutdown
	select {
	case err := <-errCh:
		loger.Log.Error("Server error", zap.Error(err))
		return err
	case <-quit:
		loger.Log.Info("Server is shutting down...")
	}

	// Создаем контекст с таймаутом для завершения всех текущих запросов
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Пытаемся корректно завершить работу сервера
	if err := srv.Shutdown(timeoutCtx); err != nil {
		loger.Log.Error("Server forced to shutdown", zap.Error(err))
	}

	// Ждем завершения всех горутин, связанных с обработкой запросов
	service.Wg.Wait()
	if err != nil {
		loger.Log.Error("Error closing database connection", zap.Error(err))
	}

	loger.Log.Info("Server shutdown completed")

	return nil
}
