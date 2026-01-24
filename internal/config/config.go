package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

type ConfigVariable struct {
	serverAddress string `env:"SERVER_ADDRESS"`
	baseURL       string `env:"BASE_URL"`
}

type Config struct {
	Host       string
	GetURLHost string
}

func CreateConfig() *Config {
	//Если указана переменная окружения, то используется она.
	var configVariable ConfigVariable
	var resultHost, resultGetURLHost string

	err := env.Parse(&configVariable)
	if err != nil {
		log.Fatal(err)
	}

	varHost := configVariable.serverAddress
	varGetURLHost := configVariable.baseURL

	//Если нет переменной окружения, но есть аргумент командной строки (флаг), то используется он.
	flagHost := flag.String("a", "", "host. default value: localhost")
	flagGetURLHost := flag.String("b", "", "host in getURL response")
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

	return &Config{
		Host:       resultHost,
		GetURLHost: resultGetURLHost,
	}
}
