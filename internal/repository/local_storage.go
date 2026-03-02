package repository

type URL struct {
	//[longURL]shortURL
	data map[string]string
}

func NewURLRepository() *URL {
	return &URL{
		data: make(map[string]string),
	}
}

// Получаем короткую URL
// Если не находим значение в мапе, то генерируем
func (u *URL) GetShortURL(longURL string) string {
	var shortURL string
	val, ok := u.data[longURL]

	// Значения нет, нужно создать новое
	if !ok {
		return ""
		// Генерируем значение из 7 символов
		//shortURL = service.GenerateRandomString(7)
		// Сохраняем в мапе
		//u.data[longURL] = shortURL

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

func (u *URL) WriteLocalURL(longURL string, shortURL string) {
	u.data[longURL] = shortURL
}
