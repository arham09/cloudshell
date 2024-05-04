package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	AllowedHostnames     []string
	Arguments            []string
	ServerAddress        string
	Command              string
	MaxBufferSizeBytes   int
	Port                 int
	ConnectionErrorLimit int16
	KeepalivePingTimeout int16
	LogFormat            string
	LogLevel             string
}

func Configuration(filePath string) (*Config, error) {
	if _, err := os.Stat(filePath); err != nil {
		return nil, err
	}

	viper.SetConfigFile(filePath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	return &config, nil
}
