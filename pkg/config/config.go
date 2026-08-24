package config

type Config struct {
	Database DBConfig     `yaml:"database"`
	Server   ServerConfig `yaml:"server"`
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
