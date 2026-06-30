package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	pb "go-url-shortener/internal/proto"
	"go-url-shortener/internal/repository"
	"go-url-shortener/internal/service"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// setupTestServer поднимает gRPC-сервер на in-memory listener
func setupTestServer(t *testing.T) (pb.ShortenerServiceClient, func()) {
	t.Helper()

	cfg := &config.Config{
		GetURLHost: "http://localhost:8080",
	}

	// Сервис без БД — работает на локальном репозитории (in-memory)
	var g errgroup.Group
	urlLocal := repository.NewURLRepository()
	svc := service.NewShortenerService(nil, nil, urlLocal, cfg, &g)

	// Поднимаем сервер на случайном свободном порту
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("не удалось открыть listener: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterShortenerServiceServer(grpcServer, NewShortenerGRPCServer(svc, cfg))

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	// Клиент
	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("не удалось подключиться: %v", err)
	}

	client := pb.NewShortenerServiceClient(conn)

	cleanup := func() {
		conn.Close()
		grpcServer.GracefulStop()
	}

	return client, cleanup
}

func TestShortenAndExpand(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	originalURL := "https://example.com/very/long/url"

	// 1. Сокращаем URL
	shortResp, err := client.ShortenURL(ctx, pb.URLShortenRequest_builder{
		Url: proto.String(originalURL),
	}.Build())
	if err != nil {
		t.Fatalf("ShortenURL вернул ошибку: %v", err)
	}
	if shortResp.GetResult() == "" {
		t.Fatal("ShortenURL вернул пустой результат")
	}
	t.Logf("Короткий URL: %s", shortResp.GetResult())

	// Извлекаем ID из короткого URL (последний сегмент после "/")
	shortURL := shortResp.GetResult()
	id := shortURL[len("http://localhost:8080/"):]

	// 2. Разворачиваем обратно
	expandResp, err := client.ExpandURL(ctx, pb.URLExpandRequest_builder{
		Id: proto.String(id),
	}.Build())
	if err != nil {
		t.Fatalf("ExpandURL вернул ошибку: %v", err)
	}

	if expandResp.GetResult() != originalURL {
		t.Errorf("ожидался %q, получен %q", originalURL, expandResp.GetResult())
	}
}

func TestShortenEmptyURL(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	emptyURL := ""
	_, err := client.ShortenURL(context.Background(), pb.URLShortenRequest_builder{
		Url: proto.String(emptyURL),
	}.Build())
	if err == nil {
		t.Fatal("ожидалась ошибка для пустого URL, но её нет")
	}
}

func TestListUserURLsNoToken(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Без metadata — должна быть ошибка Unauthenticated
	_, err := client.ListUserURLs(context.Background(), &emptypb.Empty{})
	if err == nil {
		t.Fatal("ожидалась ошибка Unauthenticated, но её нет")
	}
}

func TestListUserURLsWithToken(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Генерируем валидный токен для userID
	// Используем тот же механизм, что и в cookies
	token := makeTestToken(t, 1)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", token)

	resp, err := client.ListUserURLs(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("ListUserURLs вернул ошибку: %v", err)
	}

	// У нового пользователя URL нет — пустой список это норма
	t.Logf("Получено URL: %d", len(resp.GetUrl()))
}

func makeTestToken(t *testing.T, userID int) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cookies.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cookies.TokenExp)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(cookies.SecretKey))
	if err != nil {
		t.Fatalf("не удалось создать тестовый токен: %v", err)
	}

	return tokenString
}
