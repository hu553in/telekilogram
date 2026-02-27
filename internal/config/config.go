package config

import "github.com/caarlos0/env/v11"

type Config struct {
	Token        string  `env:"TOKEN,required,notEmpty"`
	AllowedUsers []int64 `env:"ALLOWED_USERS"`
	DBPath       string  `env:"DB_PATH"                 envDefault:"db.sqlite"`
	OpenAIAPIKey string  `env:"OPENAI_API_KEY"`
}

func LoadConfig() Config {
	var cfg Config
	env.Must(cfg, env.Parse(&cfg))
	return cfg
}
