package config

import "time"

type Config struct {
	Server   Server   `mapstructure:"server"`
	WatchDog WatchDog `mapstructure:"WatchDog"`

	Database struct {
		URL            string `mapstructure:"url"`
		MaxConnections int    `mapstructure:"max_connections"`
	} `mapstructure:"database"`

	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`
}

type Server struct {
	EnableHTTPS bool          `mapstructure:"enable_https"`
	Host        string        `mapstructure:"host"`
	Port        int           `mapstructure:"port"`
	UnixSocket  string        `mapstructure:"unix_socket"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

type WatchDog struct {
	PendingCheckInterval *time.Duration `mapstructure:"PendingCheckInterval"`
	RunningCheckInterval *time.Duration `mapstructure:"RunningCheckInterval"`
	FailedCheckInterval  *time.Duration `mapstructure:"FailedCheckInterval"`
	MaxWorkers           int            `mapstructure:"MaxWorkers"`
	OperationTimeout     time.Duration  `mapstructure:"OperationTimeout"`
}
