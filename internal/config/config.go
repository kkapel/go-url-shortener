package config

import "flag"

type Config struct {
	Host       string
	GetURLHost string
}

func CreateConfig() *Config {
	host := flag.String("a", "localhost:8080", "host. default value: localhost")
	getURLHost := flag.String("b", "localhost:8080", "host in getURL response")
	flag.Parse()

	return &Config{
		Host:       *host,
		GetURLHost: *getURLHost,
	}
}
