package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Token      string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Помилка завантаження .env файлу: %v", err)
	}

	return &Config{
		Token:      os.Getenv("BOT_TOKEN"),
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
	}
}
