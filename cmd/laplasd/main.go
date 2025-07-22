package main

import (
	"laplasd/internal/config"
	"laplasd/internal/daemon"
	"laplasd/internal/logger"

	"github.com/spf13/viper"
)

func main() {
	logger.Init()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/laplasd/")

	// Установите значения по умолчанию
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.timeout", "30s")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Log.Warn("Config file not found, using defaults")
		} else {
			logger.Log.Fatalf("Error reading config: %v", err)
		}
	}

	// Загрузите конфиг в структуру
	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		logger.Log.Fatalf("Failed to unmarshal config: %v", err)
	}

	logger.Log.Info("Laplas: Starting daemon")

	// Передайте конфиг в daemon.New
	d := daemon.New(logger.Log, &cfg)

	if err := d.Run(); err != nil {
		logger.Log.Fatalf("Daemon error: %v", err)
	}
}
