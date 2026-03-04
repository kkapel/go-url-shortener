package cookies

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const SecretKey = "testKey1" // убрать в бд
const TokenExp = time.Hour * 3

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

func RequestCookies(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("access_token")

		if err == http.ErrNoCookie {
			//Если куки нет, выдаем новую куку
			//generateCookie()
		} else {
			http.Error(w, "Ошибка при обработки куки", http.StatusBadRequest)
		}

		cookie.MaxAge = 1 // убрать
	})
}

func generateCookie() (string, error) {
	// Создаем jwt-строку
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		UserID: 1,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil

}
