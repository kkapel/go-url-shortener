package service

import (
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/repository"
	"net/http"
	"sync"
)

func GetURL(inputURL string, filePath string, URLType string, mu *sync.Mutex, databaseDsn string,
	req *http.Request, localURL *repository.URL) (string, error) {
	var URL string
	var err error
	var fileStorage []repository.URLFileStorage

	// Проверяем, как будем хранить URL
	// Очередность
	// Если заполнено поле DATABASE_DSN, вызываем БД
	// В противном случае храним записи в файле
	// В противном случае храним значения локально

	if databaseDsn != "" {
		URL, err = repository.GetURLFromDB(req.Context(), inputURL, URLType, req.Method)
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

		// Также смотрим, есть ли userID в куке access-token
		var userID int
		cookie, err := req.Cookie("access_token")

		if err != http.ErrNoCookie {
			return "", nil
		} else if err == http.ErrNoCookie {
			userID = 0
		} else {
			//получаем userID
			userID, err = cookies.GetUserID(cookie.Value)
		}

		if userID == cookies.UserIDNotFound {
			userID = 0
		}
		if err != nil {
			return "", err
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
