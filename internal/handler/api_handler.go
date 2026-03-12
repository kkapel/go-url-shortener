package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-url-shortener/internal/config"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/loger"
	"go-url-shortener/internal/repository"
	"go-url-shortener/internal/service"
	"io"
	"log"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

type Handler struct {
	Cfg *config.Config
	Rep *repository.Repsitory
	URL *repository.URL
}

type URL struct {
	URL string `json:"url"`
}

type ResultJSON struct {
	Result string `json:"result"`
}

type BatchJSON struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchJSONResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type ShortURLByUserResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (h *Handler) APIPagePost(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		fmt.Println("Метод ApiPagePost")
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}

		longURL := string(body)
		shortURL, err := service.GetURL(longURL, h.Cfg.FileStoragePath, "short", &h.Rep.Mu, h.Cfg.DBString, req, h.URL)

		if err != nil {
			var uniqueViolationError *repository.UniqueViolationError

			if errors.As(err, &uniqueViolationError) {
				res.WriteHeader(http.StatusConflict)
				res.Write([]byte(h.Cfg.GetURLHost + "/" + uniqueViolationError.LongURL))
				return
			}
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(h.Cfg.GetURLHost + "/" + shortURL))

	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) APIPageGet(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:

		loger.Log.Info("APIPageGet", zap.Any("Request Body", req.Body))

		shortURL := req.PathValue("id")
		longURL, err := service.GetURL(shortURL, h.Cfg.FileStoragePath, "long", &h.Rep.Mu, h.Cfg.DBString, req, h.URL)

		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		loger.Log.Info("APIPageGet", zap.String("shortURL", shortURL))
		loger.Log.Info("APIPageGet", zap.String("LongURL", longURL))

		res.Header().Set("content-type", "text/plain")
		res.Header().Set("Location", longURL)
		//http code 307
		res.WriteHeader(http.StatusTemporaryRedirect)

	default:
		errorResponse(res)
	}

}

func (h *Handler) APIPagePostJSON(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		var url URL
		var resultJSON ResultJSON
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		loger.Log.Info("APIPagePostJSON", zap.Any("Request Body", body))

		if err := json.Unmarshal(body, &url); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		shortURL, err := service.GetURL(url.URL, h.Cfg.FileStoragePath, "short", &h.Rep.Mu, h.Cfg.DBString, req, h.URL)

		var uniqueViolationError *repository.UniqueViolationError

		if err != nil && !errors.As(err, &uniqueViolationError) {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		//Фомрмируем ответ
		// в т.ч. для UniqueViolationError
		resultJSON.Result = h.Cfg.GetURLHost + "/" + shortURL
		loger.Log.Info("APIPagePostJSON", zap.String("short_url", shortURL))
		resp, err := json.Marshal(resultJSON)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		if uniqueViolationError != nil {
			//UniqueViolationError
			res.WriteHeader(http.StatusConflict)
		} else {
			res.WriteHeader(http.StatusCreated)
		}
		loger.Log.Info("APIPagePostJSON", zap.String("result", string(resp)))
		res.Write(resp)

	default:
		errorResponse(res)
	}
}

func (h *Handler) APIGetPing(res http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodGet:

		if err := repository.CheckConnect(); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)

	default:
		errorResponse(res)
	}

}

func (h *Handler) APIPagePostBatch(res http.ResponseWriter, req *http.Request) {
	loger.Log.Info("APIPagePostBatch starts")

	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса (JSON-массив)
		var batchJSON []BatchJSON
		var BatchJSONResponseVar []BatchJSONResponse
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		if err := json.Unmarshal(body, &batchJSON); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		loger.Log.Info("APIPagePostBatch", zap.Any("Request Body", batchJSON))

		//Получаем короткий URL
		//Проходим по циклу оригинальных(длинных) URL
		for i := range batchJSON {
			shortURL, err := service.GetURL(batchJSON[i].OriginalURL, h.Cfg.FileStoragePath, "short", &h.Rep.Mu, h.Cfg.DBString, req, h.URL)

			if err != nil {
				loger.Log.Error("Ошибка в методе GetURL", zap.String("error", err.Error()))
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}

			newBatchJSONResponse := BatchJSONResponse{
				CorrelationID: batchJSON[i].CorrelationID,
				ShortURL:      h.Cfg.GetURLHost + "/" + shortURL,
			}

			BatchJSONResponseVar = append(BatchJSONResponseVar, newBatchJSONResponse)
		}

		// Формируем ответ
		resp, err := json.Marshal(BatchJSONResponseVar)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		res.WriteHeader(http.StatusCreated)
		loger.Log.Info("APIPagePostBatch", zap.String("result", string(resp)))
		res.Write(resp)

	default:
		errorResponse(res)
	}
}

// Хэндлер получения ссылок юзера
func (h *Handler) APIPageGetUserURLs(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		// Получаем из куки UserID
		cookie, err := req.Cookie("access_token")
		if err != nil {
			if err == http.ErrNoCookie {
				res.WriteHeader(http.StatusNoContent)
				return
			} else {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
		}

		id, err := cookies.GetUserID(cookie.Value)

		loger.Log.Info("APIPageGetUserURLs", zap.Int("input id", id))

		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		// Получаем список url-ов из БД
		urls, err := repository.GetURLsByUserID(req.Context(), id)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(urls) == 0 {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		// Заполняем ответ
		var urlsResponse []ShortURLByUserResponse
		for shortURL, longURL := range urls {
			shortURLByUserResponseVar := &ShortURLByUserResponse{
				ShortURL:    h.Cfg.GetURLHost + "/" + shortURL,
				OriginalURL: longURL,
			}

			urlsResponse = append(urlsResponse, *shortURLByUserResponseVar)
		}

		resp, err := json.Marshal(urlsResponse)

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("content-type", "application/json")
		res.WriteHeader(http.StatusOK)
		loger.Log.Info("APIPageGetUserURLs", zap.String("result", string(resp)))
		res.Write(resp)

	default:
		errorResponse(res)
	}

}

func (h *Handler) APIDeleteURLs(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodDelete:
		// Читаем тело запроса (JSON-массив)
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "Cannot read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		// Получаем из куки UserID
		// Можно потом вынести в отдельный метод
		cookie, err := req.Cookie("access_token")
		if err != nil {
			if err == http.ErrNoCookie {
				res.WriteHeader(http.StatusNoContent)
				return
			} else {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
		}

		id, err := cookies.GetUserID(cookie.Value)

		loger.Log.Info("APIPageGetUserURLs", zap.Int("input id", id))

		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		var arrayURLs []string

		// Парсим
		if err := json.Unmarshal(body, &arrayURLs); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		// Взаимодействуем с БД в отдельной go-рутине
		go func(data []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			inputCh := generatorString(arrayURLs)
			fanoutCh := fanOut(inputCh)
			finalCh := fanIn(ctx, id, fanoutCh...)
			batchWorkerDelete(ctx, finalCh)

		}(arrayURLs)

		res.WriteHeader(http.StatusAccepted)

	default:
		errorResponse(res)
	}
}

func errorResponse(res http.ResponseWriter) {
	res.WriteHeader(http.StatusBadRequest)
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

func batchWorkerDelete(ctx context.Context, inputCh <-chan string) {
	var ids []string

	for id := range inputCh {
		ids = append(ids, id)

		if len(ids) >= 100 { //Устанавливаем лимит для батча - 100
			err := repository.SetDeletedFlag(ctx, ids)
			if err != nil {
				log.Printf("ошибка батч-удаления: %v", err)
			}
			ids = ids[:0]
		}

		if len(ids) > 0 {
			err := repository.SetDeletedFlag(ctx, ids)
			if err != nil {
				log.Printf("ошибка батч-удаления: %v", err)
			}
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
func fanIn(ctx context.Context, userID int, resultChs ...chan string) chan string {
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
				available, err := repository.CheckDeleteAvailable(ctx, userID, data)
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

		return finalCh
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
