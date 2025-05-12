package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Printer struct {
		Address    string `json:"address"`
		AccessCode string `json:"access_code"`
		MQTTTopic  string `json:"mqtt_topic"`
		Serial     string `json:"serial"`
		UserName   string `json:"username"`
		ClientId   string `json:"client_id"`
	}
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %s", path)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
