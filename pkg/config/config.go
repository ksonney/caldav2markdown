package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	URL      string
	Username string
	Password string
	Output   string
}

func LoadFromEnvFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config := &Config{
		Output: "./events",
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		value = strings.Trim(value, "\"'")

		switch key {
		case "CALDAV_URL":
			config.URL = value
		case "CALDAV_USERNAME":
			config.Username = value
		case "CALDAV_PASSWORD":
			config.Password = value
		case "OUTPUT_DIR":
			config.Output = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("CALDAV_URL is required")
	}
	if c.Username == "" {
		return fmt.Errorf("CALDAV_USERNAME is required")
	}
	if c.Password == "" {
		return fmt.Errorf("CALDAV_PASSWORD is required")
	}
	return nil
}