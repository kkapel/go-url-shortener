package service

import (
	"go-url-shortener/internal/repository"
)

func GetURL(inputURL string, filePath string, URLType string) (string, error) {

	// Получаем URL из файла
	// В случае отсутствия генерируем новый URL и сохраняем в файл

	URL, fileStorage, err := repository.GetURLFromFile(inputURL, filePath, URLType)

	if err != nil {
		return "", err
	}

	if URLType == "short" && URL == "" {
		URL = GenerateRandomString(7)
		repository.WriteToFile(fileStorage, URL, inputURL, filePath)
	}

	return URL, nil

}
