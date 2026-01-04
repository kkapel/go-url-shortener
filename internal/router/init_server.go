package router

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
)

func Run() {
	fmt.Println("Версия Go:", runtime.Version())
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, apiPagePost)
	mux.HandleFunc(`/{id}`, apiPageGet)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		// Возможно стоит убрать панику
		panic(err)
	}
	fmt.Println("Hello world!")
}

func apiPagePost(res http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodPost:
		// Читаем тело запроса
		body, err := io.ReadAll(req.Body)
		if err != nil {
			// Дописать обработку ошибки
			return
		}

		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte("http://localhost:8080/EwHXdJfB"))
		fmt.Println(body)

	default:

	}
}

func apiPageGet(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:

		res.Header().Set("content-type", "text/plain")
		//res.Header().Set("Location", respURL.Request.URL.String())
		//http code 307
		res.WriteHeader(http.StatusTemporaryRedirect)
		res.Write([]byte("http://localhost:8080/EwHXdJfB"))

	default:
		errorResponse(res, req)
	}

}

func errorResponse(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusBadRequest)
}
