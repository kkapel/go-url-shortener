package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

type ConfigVariable struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

type Config struct {
	Host            string
	GetURLHost      string
	FileStoragePath string
}

func CreateConfig() *Config {
	//Если указана переменная окружения, то используется она.
	var configVariable ConfigVariable
	var resultHost, resultGetURLHost, resultFileStoragePath string

	err := env.Parse(&configVariable)
	if err != nil {
		log.Fatal(err)
	}

	varHost := configVariable.ServerAddress
	varGetURLHost := configVariable.BaseURL
	varFileStorePath := configVariable.FileStoragePath

	//Если нет переменной окружения, но есть аргумент командной строки (флаг), то используется он.
	flagHost := flag.String("a", "", "host. default value: localhost")
	flagGetURLHost := flag.String("b", "", "host in getURL response")
	flagFileStoragePath := flag.String("f", "", "local file storage path")
	flag.Parse()

	if varHost != "" {
		resultHost = varHost
	} else if *flagHost != "" {
		resultHost = *flagHost
	} else {
		resultHost = "localhost:8080"
	}

	if varGetURLHost != "" {
		resultGetURLHost = varGetURLHost
	} else if *flagGetURLHost != "" {
		resultGetURLHost = *flagGetURLHost
	} else {
		resultGetURLHost = "http://localhost:8080"
	}

	if varFileStorePath != "" {
		resultFileStoragePath = varFileStorePath
	} else if *flagFileStoragePath != "" {
		resultFileStoragePath = *flagFileStoragePath
	} else {
		resultFileStoragePath = `D:\Learning\Go\go-url-shortener\go-url-shortener\FILE_STORAGE_PATH.txt`
	}

	log.Printf("%s", "Переменная varFileStorePath в функции CreateConfig: "+varFileStorePath)
	log.Printf("%s", "Переменная flagFileStoragePath в функции CreateConfig: "+*flagFileStoragePath)
	log.Printf("%s", "Переменная resultFileStoragePath в функции CreateConfig: "+resultFileStoragePath)

	return &Config{
		Host:            resultHost,
		GetURLHost:      resultGetURLHost,
		FileStoragePath: resultFileStoragePath,
	}
}
