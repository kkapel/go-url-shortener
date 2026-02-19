package config

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/caarlos0/env/v6"
)

type ConfigVariable struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DBString        string `env:"DATABASE_DSN"`
}

type Config struct {
	Host            string
	GetURLHost      string
	FileStoragePath string
	DBString        string
}

func CreateConfig() *Config {
	//Если указана переменная окружения, то используется она.
	var configVariable ConfigVariable
	var resultHost, resultGetURLHost, resultFileStoragePath, resultDBString string

	err := env.Parse(&configVariable)
	if err != nil {
		log.Fatal(err)
	}

	varHost := configVariable.ServerAddress
	varGetURLHost := configVariable.BaseURL
	varFileStorePath := configVariable.FileStoragePath
	varDBString := configVariable.DBString

	//Если нет переменной окружения, но есть аргумент командной строки (флаг), то используется он.
	flagHost := flag.String("a", "", "host. default value: localhost")
	flagGetURLHost := flag.String("b", "", "host in getURL response")
	flagFileStoragePath := flag.String("f", "", "local file storage path")
	flagDB := flag.String("d", "", "db connect string")
	flag.Parse()

	if varHost != "" {
		resultHost = varHost
	} else if *flagHost != "" {
		resultHost = *flagHost
	} else {
		resultHost = ""
	}

	if varGetURLHost != "" {
		resultGetURLHost = varGetURLHost
	} else if *flagGetURLHost != "" {
		resultGetURLHost = *flagGetURLHost
	} else {
		resultGetURLHost = ""
	}

	if varFileStorePath != "" {
		resultFileStoragePath = varFileStorePath
	} else if *flagFileStoragePath != "" {
		resultFileStoragePath = *flagFileStoragePath
	} else {
		//хардкорный путь задан согласно заданию iter9:
		//Если нет ни переменной окружения, ни флага, то используется значение по умолчанию.
		resultFileStoragePath = filepath.Join(".", "storage.txt")
	}

	//db connect
	switch {
	case varDBString != "":
		resultDBString = varDBString
	case *flagDB != "":
		resultDBString = *flagDB
	default:
		resultDBString = "localhost"
	}

	log.Printf("%s", "Переменная varFileStorePath в функции CreateConfig: "+varFileStorePath)
	log.Printf("%s", "Переменная flagFileStoragePath в функции CreateConfig: "+*flagFileStoragePath)
	log.Printf("%s", "Переменная resultFileStoragePath в функции CreateConfig: "+resultFileStoragePath)

	return &Config{
		Host:            resultHost,
		GetURLHost:      resultGetURLHost,
		FileStoragePath: resultFileStoragePath,
		DBString:        resultDBString,
	}
}
