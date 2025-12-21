package router

import (
	"fmt"
	"io"
	"net/http"
)

func Run() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, apiPage)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
	fmt.Println("Hello world!")
}

func apiPage(res http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodPost {
		// Читаем тело запроса
		body, err := io.ReadAll(req.Body)
		if err != nil {
			// Дописать обработку ошибки
			return
		}

		if string(body) == "https://practicum.yandex.ru/" {
			res.Header().Set("content-type", "text/plain")
			res.WriteHeader(http.StatusCreated)
			res.Write([]byte("http://localhost:8080/EwHXdJfB"))
		}

	}
}
