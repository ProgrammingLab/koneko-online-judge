package conf

import (
	"github.com/BurntSushi/toml"
	"github.com/gedorinku/koneko-online-judge/server/logger"
)

type Config struct {
	Koneko    KoneConfig      `toml:"Koneko"`
	Judgement JudgementConfig `toml:"Judgement"`
	Client    ClientConfig    `toml:"Client"`
}

type KoneConfig struct {
	DBHost     string `toml:"dbHost"`
	DBName     string `toml:"dbName"`
	DBUser     string `toml:"dbUser"`
	DBPassword string `toml:"dbPassword"`
	RedisHost  string `toml:"redisHost"`
}

type JudgementConfig struct {
	Concurrently int `toml:"concurrently"`
}

type ClientConfig struct {
	BasePath          string `toml:"basePath"`
	PasswordResetPath string `toml:"passwordResetPath"`
	RegistrationPath  string `toml:"registrationPath"`
}

var cfg = &Config{}

func LoadConfig() error {
	_, err := toml.DecodeFile("koneko.toml", cfg)
	if err != nil {
		logger.AppLog.Errorf("load config error: %+v", err)
	}
	return err
}

func GetConfig() *Config {
	return cfg
}
