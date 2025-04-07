package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
)

type Config struct {
	Env            string         `yaml:"env" env-default:"local"`
	PostgresCfgMac PostgresConfig `yaml:"postgres_mac"`
	PostgresCfgWin PostgresConfig `yaml:"postgres_win"`
	RedisCfg       RedisConfig    `yaml:"redis"`
	BinanceConfig  BinanceConfig  `yaml:"binance_http_client"`
}

type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Db       int    `yaml:"db"`
	Password string `yaml:"password"`
}

type BinanceConfig struct {
	BaseURL  string   `yaml:"base_url"`
	Endpoint string   `yaml:"ticker_price_endpoint"`
	Streams  []string `yaml:"streams"`
}

func MustLoad() *Config {
	path := fetchConfigPath()
	if path == "" {
		panic("config file is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file not found " + path)
	}

	var config Config

	if err := cleanenv.ReadConfig(path, &config); err != nil {
		panic("failed to read config " + err.Error())
	}

	return &config
}

func MustLoadByPath(path string) *Config {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file not found " + path)
	}

	var config Config

	if err := cleanenv.ReadConfig(path, &config); err != nil {
		panic("failed to read config " + err.Error())
	}

	return &config
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
