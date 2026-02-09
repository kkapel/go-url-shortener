package service

import (
	"go-url-shortener/internal/repository"
	"sync"
)

func GetURL(inputURL string, filePath string, URLType string, mu *sync.Mutex) (string, error) {

	// Получаем URL из файла
	// В случае отсутствия генерируем новый URL и сохраняем в файл

	URL, fileStorage, err := repository.GetURLFromFile(inputURL, filePath, URLType, mu)

	if err != nil {
		return "", err
	}

	if URLType == "short" && URL == "" {
		URL = GenerateRandomString(7)
		repository.WriteToFile(fileStorage, URL, inputURL, filePath, mu)
	}

	return URL, nil

}
