package repository

import (
	"encoding/json"
	"errors"
	"go-url-shortener/internal/service"
	"io"
	"log"
	"os"
)

type URL struct {
	//[longURL]shortURL
	data map[string]string
}

func NewURLRepository() *URL {
	return &URL{
		data: make(map[string]string),
	}
}

type URL_file_storage struct {
	UUID      uint   `json:"uuid"`
	Short_URL string `json:"short_url"`
	Long_URL  string `json:"long_url"`
}

/*

// Получаем короткую URL
// Если не находим значение в мапе, то генерируем
func (u *URL) GetShortURL(longURL string) string {
	var shortURL string
	val, ok := u.data[longURL]

	// Значения нет, нужно создать новое
	if !ok {
		// Генерируем значение из 7 символов
		shortURL = service.GenerateRandomString(7)
		// Сохраняем в мапе
		u.data[longURL] = shortURL

	} else {
		shortURL = val
	}

	return shortURL
}

func (u *URL) GetLongURL(shortURL string) string {
	var longURL string

	for key, value := range u.data {
		if value == shortURL {
			longURL = key
			break // Выходим, как только нашли первое совпадение
		}
	}

	return longURL

}

*/

// Получаем короткую URL
// Если не находим значение в файле, то генерируем

/*
		Данные хранятся в файле в формате JSON
		Пример файла:
		[
	  		{"uuid":"1","short_url":"4rSPg8ap","original_url":"http://yandex.ru"},
	  		{"uuid":"2","short_url":"edVPg3ks","original_url":"http://ya.ru"},
	  		{"uuid":"3","short_url":"dG56Hqxm","original_url":"http://practicum.yandex.ru"},
	  ...
		]
*/
func (u *URL) GetURL(input_URL string, filePath string, URLType string) (error, string) {
	var result_URL string
	var URL_file_storages []URL_file_storage
	log.Printf("Переменная filepath в функции GetURL:W" + filePath)

	// открываем файл
	// если его нет, то создаем
	flag := os.O_RDWR | os.O_CREATE | os.O_APPEND
	file, err := os.OpenFile(filePath, flag, 0666)

	if err != nil {
		return err, ""
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	errDecode := decoder.Decode(&URL_file_storages)

	if errDecode != nil && errDecode != io.EOF {
		return err, ""
	}

	// логика, если ищем shortURL
	if URLType == "short" {
		// файл не пустой
		// ищем short_url
		if errDecode != io.EOF {
			for _, line := range URL_file_storages {
				if line.Long_URL == input_URL {
					return nil, line.Short_URL
				}
			}
		}

		// Если прошлись по всему файлу и не нашли short_url или файл был пустой
		// Генерируем значение из 7 символов
		result_URL = service.GenerateRandomString(7)
		//Записываем новое значение в файл
		err = writeToFile(URL_file_storages, result_URL, input_URL, file)

		if err != nil {
			return err, ""
		}

	} else if URLType == "long" {
		// файл не пустой
		// ищем short_url
		if errDecode != io.EOF {
			for _, line := range URL_file_storages {
				if line.Short_URL == input_URL {
					result_URL = line.Long_URL
				}
			}
		}
	}

	if result_URL == "" {
		error := errors.New("Возникла ошибка при получении URL")
		return error, ""
	}

	return nil, result_URL

}

func writeToFile(URL_file_storages []URL_file_storage, shortURL string, longURL string, file *os.File) error {

	// Создадим новый элемент в JSON
	newItem := URL_file_storage{
		UUID:      1,
		Short_URL: shortURL,
		Long_URL:  longURL,
	}

	// добавим новое значение в слайс
	URL_file_storages = append(URL_file_storages, newItem)

	// очистим файл и переведем курсор
	file.Truncate(0)
	file.Seek(0, 0)

	encoder := json.NewEncoder(file)

	return encoder.Encode(URL_file_storages)
}
