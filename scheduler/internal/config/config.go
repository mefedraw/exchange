package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env        string `yaml:"env" env-default:"local"`
	HttpServer `yaml:"httpServer"`
}

type HttpServer struct {
	Address      string        `yaml:"address" env-default:"localhost:8082"`
	Timeout      time.Duration `yaml:"timeout" env-default:"4s"`
	IddleTimeout time.Duration `yaml:"iddle_timeout" env-default:"60s"`
}

func MustLoad() *Config {
	confingPath := os.Getenv("CONFIG_PATH")
	if confingPath == "" {
		log.Fatal("CONFIG_PATH environment variable not set")
	}
	if _, err := os.Stat(confingPath); os.IsNotExist(err) {
		log.Fatalf("CONFIG_PATH does not exist: %s", confingPath)
	}
	var cfg Config

	if err := cleanenv.ReadConfig(confingPath, &cfg); err != nil {
		log.Fatalf("cannot read config file: %s", err)
	}
	return &cfg
}

//type PostgresConn struct {
//	host     string `yaml:"host" env-default:"localhost"`
//	port     string `yaml:"port" env-default:"5432"`
//	user     string `yaml:"user" env-default:"postgres"`
//	password string `yaml:"password" env-default:"postgres"`
//	dbname   string `yaml:"dbname" env-default:"postgres"`
//}
