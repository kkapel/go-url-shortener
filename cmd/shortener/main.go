package main

import (
	"go-url-shortener/internal/router"
	"log"
)

func main() {
	if err := router.Run(); err != nil {
		log.Fatal(err)
	}
}
