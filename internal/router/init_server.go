package router

import (
	"go-url-shortener/internal/handler"
	"net/http"
)

func Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, handler.ApiPagePost)
	mux.HandleFunc(`/{id}`, handler.ApiPageGet)

	return http.ListenAndServe(`:8080`, mux)

}
