package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL  string
	RedisAddr    string
	RedisPass    string
	RedisDB      int
	ServerAddr   string
	PublicURL    string
	ClientURL    string
	CookieSecure bool
	WorkerCount  int
}

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	cfg := Config{}
	var err error

	if cfg.DatabaseURL, err = requiredString("DATABASE_URL", false); err != nil {
		return Config{}, err
	}
	if cfg.RedisAddr, err = requiredString("REDIS_ADDR", false); err != nil {
		return Config{}, err
	}
	if cfg.RedisPass, err = requiredString("REDIS_PASSWORD", true); err != nil {
		return Config{}, err
	}
	if cfg.RedisDB, err = requiredInt("REDIS_DB"); err != nil {
		return Config{}, err
	}
	if cfg.ServerAddr, err = requiredString("SERVER_ADDR", false); err != nil {
		return Config{}, err
	}
	if cfg.PublicURL, err = requiredString("PUBLIC_URL", false); err != nil {
		return Config{}, err
	}
	if cfg.ClientURL, err = requiredString("CLIENT_URL", false); err != nil {
		return Config{}, err
	}
	if cfg.CookieSecure, err = requiredBool("COOKIE_SECURE"); err != nil {
		return Config{}, err
	}
	if cfg.WorkerCount, err = requiredInt("WORKER_COUNT"); err != nil {
		return Config{}, err
	}

	cfg.PublicURL = trimRightSlash(cfg.PublicURL)
	cfg.ClientURL = trimRightSlash(cfg.ClientURL)

	return cfg, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return parseDotEnv(file)
}

func parseDotEnv(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf(".env:%d is missing =", lineNumber)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf(".env:%d is missing key", lineNumber)
		}

		value = strings.Trim(value, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func requiredString(key string, allowEmpty bool) (string, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return "", fmt.Errorf("%s is required", key)
	}
	value = strings.TrimSpace(value)
	if value == "" && !allowEmpty {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func requiredInt(key string) (int, error) {
	value, err := requiredString(key, false)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return parsed, nil
}

func requiredBool(key string) (bool, error) {
	value, err := requiredString(key, false)
	if err != nil {
		return false, err
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return parsed, nil
}

func trimRightSlash(value string) string {
	return strings.TrimRight(value, "/")
}
