package config

import "flag"

type Config struct {
	Host string
}

func CreateConfig() *Config {
	host := flag.String("a", ":8080", "host. default value: localhost")
	flag.Parse()

	return &Config{
		Host: *host,
	}
}
