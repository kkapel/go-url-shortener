package cookies

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const SECRET_KEY = "testKey1" // убрать в бд
const TOKEN_EXP = time.Hour * 3

type Claims struct {
	jwt.RegisteredClaims
	UserId int
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
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserId: 1,
	})

	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}

	return tokenString, nil

}
