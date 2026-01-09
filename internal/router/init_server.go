package router

import (
	"go-url-shortener/internal/handler"
	"net/http"
)

func Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, handler.APIPagePost)
	mux.HandleFunc(`/{id}`, handler.APIPageGet)

	return http.ListenAndServe(`:8080`, mux)

}
