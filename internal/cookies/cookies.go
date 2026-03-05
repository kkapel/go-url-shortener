package cookies

import (
	"context"
	"fmt"
	"go-url-shortener/internal/repository"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	userIDNotFound    = -2
	tokenIsNotValid   = -1
	tokerParsingError = -100
)

const SecretKey = "testKey1" // убрать в бд
const TokenExp = time.Hour * 3

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

func RequestCookies(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var newToken string
		var userID int
		cookie, err := r.Cookie("access_token")

		if err == http.ErrNoCookie {
			//Если куки нет, выдаем новую куку
			newToken, userID, err = generateToken(r.Context())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// другая ошибка
			// тоже выдаем куку
		} else if err != http.ErrNoCookie {
			newToken, userID, err = generateToken(r.Context())
		} else {

			// Проверяем подлинность куки
			requestUserID, err := GetUserID(cookie.Value)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Если кука присутствует в запросе, но не содержит ID пользователя, хендлер должен возвращать HTTP-статус 401
			if requestUserID == userIDNotFound {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Если не проходит проверку подлинности, выдаем новую куку
			if requestUserID == tokenIsNotValid {
				newToken, userID, err = generateToken(r.Context())

				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

		}

		// если заполнен newToken, то выдаем его пользователю в ответе
		if newToken != "" {
			cookie := &http.Cookie{
				Name:  "access_token",
				Value: newToken,
				Path:  "/",
			}
			http.SetCookie(w, cookie)

			repository.InsertUserId(r.Context(), userID, newToken)
		}

		h.ServeHTTP(w, r)

	})
}

func generateToken(ctx context.Context) (string, int, error) {
	// Создаем jwt-строку
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	userID, err := repository.GetLastUserID(ctx)

	if err != nil {
		return "", 0, err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		UserID: userID + 1,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", 0, err
	}

	return tokenString, userID + 1, nil

}

func GetUserID(tokenString string) (int, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			return []byte(SecretKey), nil
		})
	if err != nil {
		return tokerParsingError, err
	}

	if !token.Valid {
		fmt.Println("Token is not valid")
		return tokenIsNotValid, nil
	}

	//Если userID не заполнен, будет по умолчанию значение 0
	if claims.UserID < 1 {
		return userIDNotFound, nil
	}

	fmt.Println("Token is valid")
	return claims.UserID, nil
}
