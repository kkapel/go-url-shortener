package service

import (
	"math/rand/v2"
	"strings"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Генерация URL
func GenerateRandomString(n int) string {
	var sb strings.Builder
	sb.Grow(n) // Выделяем память n-байт в билдере

	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.IntN(len(charset))])
	}

	return sb.String()
}
