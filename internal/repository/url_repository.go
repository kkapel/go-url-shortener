package repository

import (
	"encoding/json"
	"errors"
	"go-url-shortener/internal/loger"
	"io"
	"os"
	"sync"

	"go.uber.org/zap"
)

type Repsitory struct {
	mu sync.RWMutex
}

type URLFileStorage struct {
	UUID     uint   `json:"uuid"`
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

func CreateRepository() *Repsitory {
	return &Repsitory{
		mu: sync.RWMutex{},
	}
}

// Возвращает URL из файла
// Если shortURL не найден, возвращается пустая строка
// Если LongURL не найден, возращается ошибка
// Возможные значения URLType: long, short
func (r *Repsitory) GetURLFromFile(inputURL string, filePath string, URLType string) (string, []URLFileStorage, error) {
	var resultURL string
	var URLFileStorages []URLFileStorage
	loger.Log.Info("GetURLFromFile", zap.String("filePath", filePath))

	// открываем файл
	// если его нет, то создаем
	r.mu.RLock()
	defer r.mu.RUnlock()

	flag := os.O_RDONLY | os.O_CREATE
	file, err := os.OpenFile(filePath, flag, 0666)

	if err != nil {
		return "", nil, err
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	errDecode := decoder.Decode(&URLFileStorages)

	if errDecode != nil && errDecode != io.EOF {
		return "", nil, err
	}

	// логика, если ищем shortURL
	switch URLType {
	case "short":
		// файл не пустой
		// ищем short_url
		if errDecode != io.EOF {
			for _, line := range URLFileStorages {
				if line.LongURL == inputURL {
					loger.Log.Info("GetURLFromFile", zap.String("нашли значение short_url", line.ShortURL))
					return line.ShortURL, URLFileStorages, nil
				}
			}
		}

	case "long":
		// файл не пустой
		// ищем long_url
		loger.Log.Info("GetURLFromFile", zap.String("Ищем в файле long_url. Исходный short_url", inputURL))
		if errDecode != io.EOF {
			for _, line := range URLFileStorages {
				if line.ShortURL == inputURL {
					resultURL = line.LongURL
					return resultURL, URLFileStorages, nil
				}
			}
		} else {
			// Для long_url отсутствие в файле считаем ошибкой
			error := errors.New("длинный url не найден")
			return "", nil, error
		}
	}

	// Если не нашли в файле, то возращаем пустое значение строки
	return "", URLFileStorages, nil

}

func (r *Repsitory) WriteToFile(URLFileStorages []URLFileStorage, shortURL string, longURL string, filePath string) error {

	r.mu.Lock()
	defer r.mu.Unlock()
	loger.Log.Info("WriteToFile",
		zap.String("Добавляем shortURL", shortURL),
		zap.String("longUrl", longURL))
	// открываем файл
	// если его нет, то создаем
	flag := os.O_RDWR | os.O_CREATE
	file, err := os.OpenFile(filePath, flag, 0666)

	if err != nil {
		return err
	}

	defer file.Close()

	//Получим новый UUID
	var UUIDNew uint
	if len(URLFileStorages) > 0 {
		UUIDNew = URLFileStorages[len(URLFileStorages)-1].UUID + 1
	}

	// Создадим новый элемент в JSON
	newItem := URLFileStorage{
		UUID:     UUIDNew,
		ShortURL: shortURL,
		LongURL:  longURL,
	}

	// добавим новое значение в слайс
	URLFileStorages = append(URLFileStorages, newItem)

	// очистим файл и переведем курсор
	file.Truncate(0)
	file.Seek(0, 0)

	encoder := json.NewEncoder(file)

	return encoder.Encode(URLFileStorages)
}
