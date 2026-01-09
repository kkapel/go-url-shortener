package handler

import (
	"fmt"
	"io"
	"net/http"
)

var LongURL1 string

func ApiPagePost(res http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		body, err := io.ReadAll(req.Body)
		if err != nil {
			// Дописать обработку ошибки
			return
		}

		LongURL1 = string(body)

		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte("http://localhost:8080/EwHXdJfB"))
		fmt.Println(body)

	default:

	}
}

func ApiPageGet(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:

		res.Header().Set("content-type", "text/plain")
		res.Header().Set("Location", LongURL1)
		//http code 307
		res.WriteHeader(http.StatusTemporaryRedirect)

	default:
		errorResponse(res, req)
	}

}

func errorResponse(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusBadRequest)
}
