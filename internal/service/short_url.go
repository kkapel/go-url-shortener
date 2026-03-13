package service

import (
	"context"
	"errors"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"
	"log"
	"sync"

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
		userID, ok := ctx.Value(cookies.UserIDKey).(int)

		if !ok {
			userID = 0
		}

		loger.Log.Info("Cookie value",
			zap.Int("userID", userID),
		)

		if userID == cookies.UserIDNotFound {
			userID = 0
		}

		if s.dbRepo != nil {
			err = s.dbRepo.InsertIntoDB(ctx, URL, inputURL, userID)
		} else if s.fileRepo != nil {
			err = s.fileRepo.WriteToFile(fileStorage, URL, inputURL, s.cfg.FileStoragePath)
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

func (s *ShortenerService) SetDeletedFlag(ctx context.Context, ids []string) error {
	return s.dbRepo.SetDeletedFlag(ctx, ids)
}

func (s *ShortenerService) CheckDeleteAvailable(ctx context.Context, userID int, shortLink string) (bool, error) {
	return s.dbRepo.CheckDeleteAvailable(ctx, userID, shortLink)
}

func (s *ShortenerService) GetLastUserID(ctx context.Context) (int, error) {
	return s.dbRepo.GetLastUserID(ctx)
}

func (s *ShortenerService) InsertUserID(ctx context.Context, userID int, newToken string) error {
	return s.dbRepo.InsertUserID(ctx, userID, newToken)
}

// generator функция для массива строк
func generatorString(input []string) chan string {
	inputCh := make(chan string)

	go func() {
		defer close(inputCh)

		for _, data := range input {
			inputCh <- data
		}
	}()

	return inputCh
}

func (s *ShortenerService) batchWorkerDelete(ctx context.Context, inputCh <-chan string) {
	var ids []string

	for id := range inputCh {
		ids = append(ids, id)

		if len(ids) >= 100 { //Устанавливаем лимит для батча - 100
			err := s.dbRepo.SetDeletedFlag(ctx, ids)
			if err != nil {
				log.Printf("ошибка батч-удаления: %v", err)
			}
			ids = ids[:0]
		}
	}

	if len(ids) > 0 {
		err := s.SetDeletedFlag(ctx, ids)
		if err != nil {
			log.Printf("ошибка батч-удаления: %v", err)
		}
	}

}

// fanOut принимает канал данных, порождает 10 горутин
func fanOut(inputCh chan string) []chan string {
	// количество горутин
	numWorkers := 10
	// каналы, в которые отправляются результаты
	channels := make([]chan string, numWorkers)

	for i := 0; i < numWorkers; i++ {
		// отправляем в слайс каналов
		channels[i] = inputCh
	}

	// возвращаем слайс каналов
	return channels
}

// fanIn объединяет несколько каналов resultChs в один.
func (s *ShortenerService) fanIn(ctx context.Context, userID int, resultChs ...chan string) chan string {
	// конечный выходной канал в который отправляем данные из всех каналов из слайса, назовём его результирующим
	finalCh := make(chan string)

	// понадобится для ожидания всех горутин
	var wg sync.WaitGroup

	// перебираем все входящие каналы
	for _, ch := range resultChs {
		// в горутину передавать переменную цикла нельзя, поэтому делаем так
		chClosure := ch

		// инкрементируем счётчик горутин, которые нужно подождать
		wg.Add(1)

		go func() {
			// откладываем сообщение о том, что горутина завершилась
			defer wg.Done()

			// получаем данные из канала
			for data := range chClosure {
				available, err := s.dbRepo.CheckDeleteAvailable(ctx, userID, data)
				if err != nil {
					// Логируем ошибку, но не роняем весь конвейер
					log.Printf("ошибка проверки ссылки %s: %v", data, err)
					continue
				}

				// Если проверка прошла, то пишем в итоговый канал
				if available {
					finalCh <- data
				}
			}
		}()
	}

	go func() {
		// ждём завершения всех горутин
		wg.Wait()
		// когда все горутины завершились, закрываем результирующий канал
		close(finalCh)
	}()

	// возвращаем результирующий канал
	return finalCh
}

func (s *ShortenerService) DeleteURLs(id int, data []string) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputCh := generatorString(data)
	fanoutCh := fanOut(inputCh)
	finalCh := s.fanIn(ctx, id, fanoutCh...)
	s.batchWorkerDelete(ctx, finalCh)

}
