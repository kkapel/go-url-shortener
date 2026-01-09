package service

import (
	"math/rand/v2"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 7) // Создаем срез из 7 пустых байт

	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}

	return string(b)
}
