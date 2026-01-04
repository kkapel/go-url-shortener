package repository

func GetURL(url string, returnShortURL bool) string {
	var result string

	// Объявим маппу с ссылками
	// map[shortUrl]LongUrl
	urlMapShort := make(map[string]string)
	urlMapShort["EwHXdJfB"] = "http://ljxwukxxrqp26.com"

	// map[LongUrl]ShortUrl
	urlMapLong := make(map[string]string)
	urlMapLong["https://practicum.yandex.ru/"] = "EwHXdJfB"

	if returnShortURL {
		result = urlMapLong[url]
		return result

	} else {
		result = urlMapShort[url]
		return result
	}
}
