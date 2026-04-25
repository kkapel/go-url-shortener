package cookies

import (
	"context"
	"errors"
	"go-url-shortener/internal/loger"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

var (
	ErrUserIDNotFound  = errors.New("UserIDNotFound")
	ErrTokenIsNotValid = errors.New("TokenIsNotValid")
	TokenParsingError  = errors.New("TokenParsingError")
)

const SecretKey = "testKey1" // убрать в бд
const TokenExp = time.Hour * 3

type userIDType string

const userIDKey userIDType = "userID"

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

type UserProvider interface {
	GetLastUserID(ctx context.Context) (int, error)
	InsertUserID(ctx context.Context, id int, token string) error
}

type Cookie struct {
	provider UserProvider
}

func NewCookie(p UserProvider) *Cookie {
	return &Cookie{provider: p}
}

func (cookieStruct *Cookie) RequestCookies(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var newToken string
		var userID int
		cookie, err := r.Cookie("access_token")

		if err == http.ErrNoCookie {
			//Если куки нет, выдаем новую куку
			newToken, userID, err = cookieStruct.generateToken(r.Context())
			if err != nil {
				loger.Log.Error("cookies.go", zap.String("Function RequestCookies", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// другая ошибка
			// тоже выдаем куку
		} else if err != http.ErrNoCookie {
			newToken, userID, err = cookieStruct.generateToken(r.Context())
			if err != nil {
				loger.Log.Error("cookies.go", zap.String("Function RequestCookies", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {

			// Проверяем подлинность куки
			_, err := GetUserID(cookie.Value)

			// Если кука присутствует в запросе, но не содержит ID пользователя, хендлер должен возвращать HTTP-статус 401
			if err == ErrUserIDNotFound {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Если не проходит проверку подлинности, выдаем новую куку
			if err == ErrTokenIsNotValid {
				newToken, userID, err = cookieStruct.generateToken(r.Context())

				if err != nil {
					loger.Log.Error("cookies.go", zap.String("Function RequestCookies", err.Error()))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			if err != nil {
				loger.Log.Error("cookies.go", zap.String("Function RequestCookies", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
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

			cookieStruct.provider.InsertUserID(r.Context(), userID, newToken)
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		h.ServeHTTP(w, r.WithContext(ctx))

	})
}

func (cookieStruct *Cookie) generateToken(ctx context.Context) (string, int, error) {
	// Создаем jwt-строку
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	userID, err := cookieStruct.provider.GetLastUserID(ctx)

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
		return 0, TokenParsingError
	}

	if !token.Valid {
		loger.Log.Info("cookies.go", zap.String("Func GetUserID", "Token is not valid"))

		return 0, ErrTokenIsNotValid
	}

	//Если userID не заполнен, будет по умолчанию значение 0
	if claims.UserID < 1 {
		return 0, ErrUserIDNotFound
	}

	loger.Log.Info("cookies.go", zap.String("Func GetUserID", "Token is valid"))
	return claims.UserID, nil
}

func GetUserValue(ctx context.Context) (int, error) {
	val := ctx.Value(userIDKey)
	if val == nil {
		return 0, nil
	}

	userID, ok := val.(int)

	if !ok {
		return 0, errors.New("Invalid userID in context")
	}

	return userID, nil
}
