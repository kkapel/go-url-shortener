package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/caarlos0/env/v6"
)

type ConfigVariable struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DBString        string `env:"DATABASE_DSN"`
	FlagAuditFile   string `env:"AUDIT_FILE"`
	FlagAuditURL    string `env:"AUDIT_URL"`
	EnableHttps     bool   `env:"ENABLE_HTTPS"`
	ConfigVariable  string `env:"CONFIG"` // Имя файла конфигурации
}

type FileConfigVariable struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DBString        string `json:"database_dsn"`
	FlagAuditFile   string `json:"audit_file"`
	FlagAuditURL    string `json:"audit_url"`
	EnableHttps     bool   `json:"enable_https"`
}

type Config struct {
	Host            string
	GetURLHost      string
	FileStoragePath string
	DBString        string
	FlagAuditFile   string
	FlagAuditURL    string
	EnableHttps     bool
}

func CreateConfig() *Config {
	//Если указана переменная окружения, то используется она.
	var configVariable ConfigVariable
	var resultHost, resultGetURLHost, resultFileStoragePath, resultDBString, resultAuditFile, resultAuditURL string

	err := env.Parse(&configVariable)
	if err != nil {
		log.Fatal(err)
	}

	varHost := configVariable.ServerAddress
	varGetURLHost := configVariable.BaseURL
	varFileStorePath := configVariable.FileStoragePath
	varDBString := configVariable.DBString
	varAuditFile := configVariable.FlagAuditFile
	varAuditURL := configVariable.FlagAuditURL
	varEnableHttps := configVariable.EnableHttps

	//Если нет переменной окружения, но есть аргумент командной строки (флаг), то используется он.
	flagHost := flag.String("a", "", "host. default value: localhost")
	flagGetURLHost := flag.String("b", "", "host in getURL response")
	flagFileStoragePath := flag.String("f", "", "local file storage path")
	flagDB := flag.String("d", "", "db connect string")
	flagAuditFile := flag.String("audit-file", "", "audit file path")
	flagAuditURL := flag.String("audit-url", "", "audit url path")
	flagEnableHttps := flag.Bool("s", false, "enable https")
	flagConfig := flag.String("c", "", "config file path")
	flag.StringVar(flagConfig, "config", "", "config file path")

	flag.Parse()

	// Получаем значения из файла конфигурации, если он указан
	var fileConfig *FileConfigVariable
	if *flagConfig != "" {
		fileConfig, err = LoadConfigFromFile(*flagConfig)
		if err != nil {
			log.Fatalf("Error loading config from file: %v", err)
		}
	}

	if varHost != "" {
		resultHost = varHost
	} else if *flagHost != "" {
		resultHost = *flagHost
	} else if fileConfig != nil && fileConfig.ServerAddress != "" {
		resultHost = fileConfig.ServerAddress
	} else {
		resultHost = "localhost:8080"
	}

	if varGetURLHost != "" {
		resultGetURLHost = varGetURLHost
	} else if *flagGetURLHost != "" {
		resultGetURLHost = *flagGetURLHost
	} else if fileConfig != nil && fileConfig.BaseURL != "" {
		resultGetURLHost = fileConfig.BaseURL
	} else {
		resultGetURLHost = "http://localhost:8080"
	}

	if varFileStorePath != "" {
		resultFileStoragePath = varFileStorePath
	} else if *flagFileStoragePath != "" {
		resultFileStoragePath = *flagFileStoragePath
	} else if fileConfig != nil && fileConfig.FileStoragePath != "" {
		resultFileStoragePath = fileConfig.FileStoragePath
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
	case fileConfig != nil && fileConfig.DBString != "":
		resultDBString = fileConfig.DBString
	default:
		resultDBString = ""
	}

	// audit file
	switch {
	case varAuditFile != "":
		resultAuditFile = varAuditFile
	case *flagAuditFile != "":
		resultAuditFile = *flagAuditFile
	case fileConfig != nil && fileConfig.FlagAuditFile != "":
		resultAuditFile = fileConfig.FlagAuditFile
	default:
		resultAuditFile = ""
	}

	// audit url
	switch {
	case varAuditURL != "":
		resultAuditURL = varAuditURL
	case *flagAuditURL != "":
		resultAuditURL = *flagAuditURL
	case fileConfig != nil && fileConfig.FlagAuditURL != "":
		resultAuditURL = fileConfig.FlagAuditURL
	default:
		resultAuditURL = ""
	}

	log.Printf("%s", "Переменная varFileStorePath в функции CreateConfig: "+varFileStorePath)
	log.Printf("%s", "Переменная flagFileStoragePath в функции CreateConfig: "+*flagFileStoragePath)
	log.Printf("%s", "Переменная resultFileStoragePath в функции CreateConfig: "+resultFileStoragePath)
	log.Printf("%s", "Переменная resultAuditFile в функции CreateConfig: "+resultAuditFile)
	log.Printf("%s", "Переменная resultAuditURL в функции CreateConfig: "+resultAuditURL)
	log.Printf("%s", "Переменная EnableHttps в функции CreateConfig: "+strconv.FormatBool(varEnableHttps || *flagEnableHttps))

	return &Config{
		Host:            resultHost,
		GetURLHost:      resultGetURLHost,
		FileStoragePath: resultFileStoragePath,
		DBString:        resultDBString,
		FlagAuditFile:   resultAuditFile,
		FlagAuditURL:    resultAuditURL,
		EnableHttps:     varEnableHttps || *flagEnableHttps,
	}
}

// LoadConfigFromFile загружает конфигурацию из указанного JSON-файла.
func LoadConfigFromFile(filePath string) (*FileConfigVariable, error) {
	// Открываем файл конфигурации
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Декодируем JSON из файла в структуру FileConfigVariable
	var config FileConfigVariable
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
