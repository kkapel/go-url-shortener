package service

import (
	"context"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"

	"go.uber.org/zap"
)

type ShortenerService struct {
	dbRepo    *repository.DB
	fileRepo  *repository.Repsitory
	localRepo *repository.URL
	cfg       *config.Config
}

func NewShortenerService(db *repository.DB, file *repository.Repsitory, localRepo *repository.URL, cfg *config.Config) *ShortenerService {
	s := &ShortenerService{
		dbRepo:    db,
		fileRepo:  file,
		localRepo: localRepo,
		cfg:       cfg,
	}

	return s
}

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
		URL, fileStorage, err = s.fileRepo.GetURLFromFile(inputURL, filePath, URLType, mu)
	} else {
		if URLType == "short" {
			URL = localURL.GetShortURL(inputURL)
		} else if URLType == "long" {
			URL = localURL.GetLongURL(inputURL)
		}
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
		userID, ok := req.Context().Value(cookies.UserIDKey).(int)

		if !ok {
			userID = 0
		}

		loger.Log.Info("Cookie value",
			zap.Int("userID", userID),
		)

		if userID == cookies.UserIDNotFound {
			userID = 0
		}

		if databaseDsn != "" {
			err = repository.InsertIntoDB(req.Context(), URL, inputURL, userID)
		} else if filePath != "" {
			repository.WriteToFile(fileStorage, URL, inputURL, filePath, mu)
		} else {
			localURL.WriteLocalURL(inputURL, URL)
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

func (s *ShortenerService) SetDeletedFlag(ctx context.Context, ids []string) error {
	return s.dbRepo.SetDeletedFlag(ctx, ids)
}

func (s *ShortenerService) CheckDeleteAvailable(ctx context.Context, userID int, shortLink string) (bool, error) {
	return s.dbRepo.CheckDeleteAvailable(ctx, userID, shortLink)
}
