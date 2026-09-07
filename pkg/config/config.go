package config

import "time"

type Config struct {
	Database DBConfig     `yaml:"database"`
	Server   ServerConfig `yaml:"server"`
	Rabbit   RabbitConfig `yaml:"rabbit"`
}
type RabbitConfig struct {
	Host     string `json:"host" yaml:"host" env:"RABBIT_HOST" env-default:"localhost"`
	Port     int    `json:"port" yaml:"port" env:"RABBIT_PORT" env-default:"5672"`
	User     string `json:"user" yaml:"user" env:"RABBIT_USER" env-default:"guest"`
	Password string `json:"password" yaml:"password" env:"RABBIT_PASSWORD" env-default:"guest"`
	VHost    string `json:"vhost" yaml:"vhost" env:"RABBIT_VHOST" env-default:"/"`

	// Параметры устойчивости соединения
	ConnectionTimeout time.Duration `json:"connection_timeout" yaml:"connection_timeout" env:"RABBIT_CONN_TIMEOUT" env-default:"10s"`
	ReconnectDelay    time.Duration `json:"reconnect_delay" yaml:"reconnect_delay" env:"RABBIT_RECONNECT_DELAY" env-default:"5s"`
	Heartbeat         time.Duration `json:"heartbeat" yaml:"heartbeat" env:"RABBIT_HEARTBEAT" env-default:"10s"`
}
type DBConfig struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	Username string `yaml:"username" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-default:"postgres"`
	DBName   string `yaml:"dbname" env:"DB_NAME" env-default:"postgres"`
	SSLMode  string `yaml:"sslmode" env:"DB_SSLMODE" env-default:"disable"`
}

type ServerConfig struct {
	Port int `yaml:"port" envconfig:"PORT" default:"8080"`
}
