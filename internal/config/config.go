package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const defaultMaxWorkers = 3

type Config struct {
	BotToken             string
	MaxWorkers           int
	InstagramCookiesFile string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"erro ao carregar arquivo .env: %w",
			err,
		)
	}

	botToken := strings.TrimSpace(
		os.Getenv("BOT_TOKEN"),
	)

	if botToken == "" {
		return nil, errors.New(
			"BOT_TOKEN não foi definido",
		)
	}

	maxWorkers, err := loadMaxWorkers()
	if err != nil {
		return nil, err
	}

	return &Config{
		BotToken:             botToken,
		MaxWorkers:           maxWorkers,
		InstagramCookiesFile: strings.TrimSpace(os.Getenv("INSTAGRAM_COOKIES_FILE")),
	}, nil
}

func loadMaxWorkers() (int, error) {
	value := strings.TrimSpace(
		os.Getenv("MAX_WORKERS"),
	)

	if value == "" {
		return defaultMaxWorkers, nil
	}

	workers, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf(
			"MAX_WORKERS inválido: %w",
			err,
		)
	}

	if workers < 1 {
		return 0, errors.New(
			"MAX_WORKERS precisa ser maior que zero",
		)
	}

	return workers, nil
}
