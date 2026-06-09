package main

import (
	"fmt"
	"go-url-shortener/internal/router"
	"log"
)

// Дефлотные значения, присваевыемые переменным уровня пакета при их объявлении, могут быть перезаписаны на этапе компиляции
// с помощью флагов -ldflags

var (
	buildVersion string = "N/A" // Значение по умолчанию для версии сборки
	buildDate    string = "N/A" // Значение по умолчанию для даты сборки
	buildCommit  string = "N/A" // Значение по умолчанию для коммита сборки
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
