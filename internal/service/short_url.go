package service

import (
	"context"
	"errors"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Stats struct {
	URLs  int // Количество сокращенных URL
	Users int // Количество пользователей в сервисе
}

type ShortenerService struct {
	ctx       context.Context
	dbRepo    *repository.DB
	fileRepo  *repository.Repsitory
	localRepo *repository.URL
	cfg       *config.Config
	group     *errgroup.Group
}

func NewShortenerService(ctx context.Context, db *repository.DB, file *repository.Repsitory, localRepo *repository.URL, cfg *config.Config, group *errgroup.Group) *ShortenerService {
	s := &ShortenerService{
		ctx:       ctx,
		dbRepo:    db,
		fileRepo:  file,
		localRepo: localRepo,
		cfg:       cfg,
		group:     group,
	}

	return s
}

var ErrConflict = errors.New("url already exists")

func (s *ShortenerService) GetURL(ctx context.Context, inputURL string, URLType string, action string) (string, error) {
	var URL string
	var err error
	var fileStorage []repository.URLFileStorage

	// Проверяем, как будем хранить URL
	// Очередность
	// Если заполнено поле DATABASE_DSN, вызываем БД
	// В противном случае храним записи в файле
	// В противном случае храним значения локально

	if s.dbRepo != nil {
		URL, err = s.dbRepo.GetURLFromDB(ctx, inputURL, URLType, action)
	} else if s.fileRepo != nil {
		URL, fileStorage, err = s.fileRepo.GetURLFromFile(inputURL, s.cfg.FileStoragePath, URLType)
	} else {
		if URLType == "short" {
			URL = s.localRepo.GetShortURL(inputURL)
		} else if URLType == "long" {
			URL = s.localRepo.GetLongURL(inputURL)
		}
	}

	var uniqueErr *repository.UniqueViolationError
	if errors.As(err, &uniqueErr) {
		// Возвращаем результат И специальную ошибку сервиса
		return uniqueErr.LongURL, ErrConflict
	}

	if err != nil {

		return "", err
	}

	// В случае отсутствия генерируем новый URL
	if URLType == "short" && URL == "" {
		URL = GenerateRandomString(7)

		// Также смотрим, есть ли userID в контексте
		var userID int

		//получаем userID
		userID, err = cookies.GetUserValue(ctx)

		if err == cookies.ErrUserIDNotFound {
			userID = 0
		} else if err != nil {
			return "", err
		}

		loger.Log.Info("Cookie value",
			zap.Int("userID", userID),
		)

		if s.dbRepo != nil {
			err = s.dbRepo.InsertIntoDB(ctx, URL, inputURL, userID)
			if err != nil {
				return "", err
			}
		} else if s.fileRepo != nil {
			err = s.fileRepo.WriteToFile(fileStorage, URL, inputURL, s.cfg.FileStoragePath)
			if err != nil {
				return "", err
			}
		} else {
			s.localRepo.WriteLocalURL(inputURL, URL)
		}

	}

	if err != nil {
		return "", err
	}

	return URL, nil

}

func (s *ShortenerService) CheckFlagDeleteExists(ctx context.Context, shortURL string) (bool, error) {
	return s.dbRepo.CheckFlagDeleteExists(ctx, shortURL)
}

func (s *ShortenerService) CheckConnect(context.Context) error {
	return s.dbRepo.CheckConnect()
}

func (s *ShortenerService) GetURLsByUserID(ctx context.Context, userID int) (map[string]string, error) {
	return s.dbRepo.GetURLsByUserID(ctx, userID)
}

func (s *ShortenerService) SetDeletedFlag(ctx context.Context, ids []string, id int) error {
	return s.dbRepo.SetDeletedFlag(ctx, ids, id)
}

func (s *ShortenerService) GetNextUserID(ctx context.Context) (int, error) {
	return s.dbRepo.GetNextUserID(ctx)
}

func (s *ShortenerService) InsertUserID(ctx context.Context, userID int, newToken string) error {
	return s.dbRepo.InsertUserID(ctx, userID, newToken)
}

func (s *ShortenerService) batchWorkerDelete(ctx context.Context, data []string, userID int) {
	err := s.dbRepo.SetDeletedFlag(ctx, data, userID)
	if err != nil {
		loger.Log.Error("final batch delete error", zap.Error(err), zap.Int("user_id", userID))
	}
}

func (s *ShortenerService) DeleteURLs(id int, data []string) {

	s.group.Go(func() error {
		s.batchWorkerDelete(s.ctx, data, id)
		return nil
	})

}

func (s *ShortenerService) GetStats(ctx context.Context) (Stats, error) {
	urls, users, err := s.dbRepo.GetStats(ctx)
	if err != nil {
		return Stats{}, err
	}
	return Stats{URLs: urls, Users: users}, nil
}
