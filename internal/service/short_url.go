package service

import (
	"go-url-shortener/internal/repository"
	"net/http"
	"sync"
)

func GetURL(inputURL string, filePath string, URLType string, mu *sync.Mutex, databaseDsn string,
	db *repository.DB, req *http.Request, localURL *repository.URL) (string, error) {
	var URL string
	var err error
	var fileStorage []repository.URLFileStorage

	// Проверяем, как будем хранить URL
	// Очередность
	// Если заполнено поле DATABASE_DSN, вызываем БД
	// В противном случае храним записи в файле
	// В противном случае храним значения локально

	if databaseDsn != "" {
		URL, err = db.GetURLFromDB(req.Context(), inputURL, URLType)
	} else if filePath != "" {
		URL, fileStorage, err = repository.GetURLFromFile(inputURL, filePath, URLType, mu)
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
		if databaseDsn != "" {
			err = db.InsertIntoDB(req.Context(), URL, URLType)
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
