package grpc

import (
	"context"
	"errors"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	pb "go-url-shortener/internal/proto"
	"go-url-shortener/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ShortenerGRPCServer struct {
	pb.UnimplementedShortenerServiceServer
	// сюда передадим сервис с бизнес-логикой
	service *service.ShortenerService
	cfg     *config.Config
}

func (s *ShortenerGRPCServer) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	// Аналог с методом APIPagePostJSON в internal/handler/handler.go

	longURL := req.GetUrl()

	if longURL == "" {
		return nil, status.Error(codes.InvalidArgument, "url should not be empty")
	}

	shortURL, err := s.service.GetURL(ctx, longURL, "short", "Post")

	if err != nil && !errors.Is(err, service.ErrConflict) {
		return nil, status.Errorf(codes.Internal, "ошибка при сокращении URL: %v", err)
	}

	result := s.cfg.GetURLHost + "/" + shortURL

	// При конфликте возвращаем код AlreadyExists
	if errors.Is(err, service.ErrConflict) {
		return nil, status.Error(codes.AlreadyExists, result)
	}

	return &pb.URLShortenResponse{
		Result: &result,
	}, nil

}

func (s *ShortenerGRPCServer) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	// Аналог с методом APIPageGet в internal/handler/handler.go
	// Здесь мы вызываем метод сервиса для получения длинного URL
	// и возвращаем результат в формате gRPC-ответа

	shortURL := req.GetId()

	// Проверяем, что входной URL не пустой
	if shortURL == "" {
		return nil, status.Error(codes.InvalidArgument, "Url should not be empty")
	}

	// Проверяем, удален ли url
	deleted, err := s.service.CheckFlagDeleteExists(ctx, shortURL)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error checking URL deletion status")
	}
	// URL удален
	if deleted {
		return nil, status.Error(codes.NotFound, "URL not found")
	}

	// Получаем длинный URL
	longURL, err := s.service.GetURL(ctx, shortURL, "long", "Get")

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка получения URL: %v", err)
	}

	return &pb.URLExpandResponse{
		Result: proto.String(longURL),
	}, nil

}

func (s *ShortenerGRPCServer) ListUserURLs(ctx context.Context, in *emptypb.Empty) (*pb.UserURLsResponse, error) {

	//Получаем токен из metadata
	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata not provided")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization token not provided")
	}

	// Извлекаем userID из JWT-токена
	userID, err := cookies.GetUserID(values[0])

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	// Получаем список URL пользователя
	urls, err := s.service.GetURLsByUserID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка получения URL: %v", err)
	}

	// Формируем ответ
	var urlData []*pb.URLData
	for shortURL, longURL := range urls {
		short := s.cfg.GetURLHost + "/" + shortURL
		long := longURL
		urlData = append(urlData, &pb.URLData{
			ShortUrl:    &short,
			OriginalUrl: &long,
		})
	}

	return &pb.UserURLsResponse{Url: urlData}, nil

}

func NewShortenerGRPCServer(svc *service.ShortenerService, cfg *config.Config) *ShortenerGRPCServer {
	return &ShortenerGRPCServer{service: svc, cfg: cfg}
}
