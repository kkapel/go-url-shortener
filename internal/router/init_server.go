package router

import (
	"context"
	"go-url-shortener/internal/audit"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/encoding"
	"go-url-shortener/internal/handler"
	"go-url-shortener/internal/ip"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"
	serverGRPC "go-url-shortener/internal/server_grpc"
	"go-url-shortener/internal/service"
	"net"
	"net/http"
	"sync"
	"time"

	pb "go-url-shortener/internal/proto"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func Run(ctx context.Context) error {
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

	// Используем errgroup для управления горутинами и их ошибками
	g, gCtx := errgroup.WithContext(ctx)

	service := service.NewShortenerService(databaseInstance, fileRepo, urlLocal, cfg, g)
	cookie := cookies.NewCookie(service)

	// Стартуем grpc
	grpc_server, err := startGRPCServer(service, cfg)
	if err != nil {
		return err
	}

	// Закрываем БД-соединение
	if databaseInstance != nil {
		defer func() { _ = databaseInstance.Close() }()
	}

	defer grpc_server.GracefulStop()

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

	r.Group(func(r chi.Router) {
		r.Use(ip.CheckIP(cfg.TrustedSubnet))
		r.Get("/api/internal/stats", h.APIGetStats)
	})

	g.Go(func() error {
		// Запускаем обработку аудита в отдельной горутине
		// Закрытие канала будет происходить после завершения работы сервера
		audit.ProcessAudit(auditChan, cfg.FlagAuditFile, auditMU, cfg.FlagAuditURL)
		return nil
	})

	errCh := make(chan error, 1)
	// Запускаем сервер в отдельной горутине
	g.Go(func() error {
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
		return nil
	})

	loger.Log.Info("Server started successfully")

	// Логика для graceful shutdown
	// Ожидаем сигнал для graceful shutdown
	select {
	case err := <-errCh:
		loger.Log.Error("Server error", zap.Error(err))
		return err
	case <-gCtx.Done():
		loger.Log.Info("Server is shutting down...")
	}

	// Создаем контекст с таймаутом для завершения всех текущих запросов
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Пытаемся корректно завершить работу сервера
	if err := srv.Shutdown(timeoutCtx); err != nil {
		loger.Log.Error("Server forced to shutdown", zap.Error(err))
	}

	// Закрываем канал аудита, чтобы завершить горутину ProcessAudit
	close(auditChan)

	// Ждем завершения всех горутин, связанных с обработкой запросов
	if err := g.Wait(); err != nil {
		return err
	}

	loger.Log.Info("Server shutdown completed")

	return nil
}

// Функция для запуска grpc-сервера
func startGRPCServer(svc *service.ShortenerService, cfg *config.Config) (*grpc.Server, error) {
	// 1. Открываем listener на нужном порту
	listener, err := net.Listen("tcp", cfg.Grpc)
	if err != nil {
		return nil, err
	}

	// Загружаем TLS-сертификаты
	creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
	if err != nil {
		return nil, err
	}

	// 2. Создаём экземпляр gRPC-сервера
	grpcServer := grpc.NewServer(grpc.Creds(creds))

	// 3. Создаём свой сервер с бизнес-логикой
	srv := serverGRPC.NewShortenerGRPCServer(svc, cfg)

	// 4. Регистрируем его в gRPC-сервере
	pb.RegisterShortenerServiceServer(grpcServer, srv)

	// 5. Запускаем в горутине, чтобы не блокировать main
	go func() {
		loger.Log.Info("Grpc server starts")
		if err := grpcServer.Serve(listener); err != nil {
			loger.Log.Error("ошибка работы gRPC-сервера", zap.Error(err))
		}
	}()

	return grpcServer, nil
}
