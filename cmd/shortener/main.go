package main

import (
	"fmt"
	"go-url-shortener/internal/router"
	"log"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	// stdout для отображения информации о сборке при запуске приложения
	fmt.Println("Build version:", valueOrNA(buildVersion))
	fmt.Println("Build date:", valueOrNA(buildDate))
	fmt.Println("Build commit:", valueOrNA(buildCommit))

	if err := router.Run(); err != nil {
		log.Fatal(err)
	}
}

func valueOrNA(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}
