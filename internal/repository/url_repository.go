package repository

func GetUrl(url string, returnShortUrl bool) string {
	var result string

	// Объявим маппу с ссылками
	// map[shortUrl]LongUrl
	urlMapShort := make(map[string]string)
	urlMapShort["EwHXdJfB"] = "https://practicum.yandex.ru/"

	// map[LongUrl]ShortUrl
	urlMapLong := make(map[string]string)
	urlMapLong["https://practicum.yandex.ru/"] = "EwHXdJfB"

	if returnShortUrl {
		result, _ = urlMapLong[url]
		return result

	} else {
		result, _ = urlMapShort[url]
		return result
	}
}
