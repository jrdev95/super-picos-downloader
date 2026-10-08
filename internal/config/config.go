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
	ReduceMedia          bool
	LowMemory            bool
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

	lowMemory, err := loadLowMemory()
	if err != nil {
		return nil, err
	}
	maxWorkers, err := loadMaxWorkers()
	if err != nil {
		return nil, err
	}
	if lowMemory {
		maxWorkers = 1
	}

	reduceMedia, err := loadMediaReduction()
	if err != nil {
		return nil, err
	}
	return &Config{
		BotToken:             botToken,
		MaxWorkers:           maxWorkers,
		LowMemory:            lowMemory,
		ReduceMedia:          reduceMedia,
		InstagramCookiesFile: strings.TrimSpace(os.Getenv("INSTAGRAM_COOKIES_FILE")),
	}, nil
}

func loadLowMemory() (bool, error) {
	value := strings.TrimSpace(os.Getenv("LOW_MEMORY"))
	if value == "" {
		return false, nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("LOW_MEMORY inválido: use true ou false: %w", err)
	}
	return enabled, nil
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

func loadMediaReduction() (bool, error) {
	value := strings.TrimSpace(os.Getenv("REDUCE_MEDIA"))
	if value == "" {
		return false, nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("REDUCE_MEDIA inválido: use true ou false: %w", err)
	}
	return enabled, nil
}
