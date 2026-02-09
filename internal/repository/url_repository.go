package repository

import (
	"encoding/json"
	"errors"
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

type URLFileStorage struct {
	UUID     uint   `json:"uuid"`
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
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

// Возвращает URL из файла
// Если shortURL не найден, возвращается пустая строка
// Если LongURL не найден, возращается ошибка
// Возможные значения URLType: long, short
func GetURLFromFile(inputURL string, filePath string, URLType string) (string, []URLFileStorage, error) {
	var resultURL string
	var URLFileStorages []URLFileStorage
	log.Printf("%s", "Переменная filepath в функции GetURLFromFile"+filePath)

	// открываем файл
	// если его нет, то создаем
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
					log.Printf("%s", "нашли значение short_url в функции GetURLFromFile: "+line.ShortURL)
					return line.ShortURL, URLFileStorages, nil
				}
			}
		}

	case "long":
		// файл не пустой
		// ищем long_url
		log.Printf("%s", "функции GetURLFromFile. Ищем в файле long_url. Исходный short_url: "+inputURL)
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

func WriteToFile(URLFileStorages []URLFileStorage, shortURL string, longURL string, filePath string) error {

	log.Printf("%s", "функция WriteToFile. Добавляем shortURL: "+shortURL+" long_url: "+longURL)
	// открываем файл
	// если его нет, то создаем
	flag := os.O_RDWR | os.O_CREATE | os.O_APPEND
	file, err := os.OpenFile(filePath, flag, 0666)

	if err != nil {
		return err
	}

	defer file.Close()

	// Создадим новый элемент в JSON
	newItem := URLFileStorage{
		UUID:     1,
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
