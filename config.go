package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Namespace *string     `json:"namespace"`
	Template  *string     `json:"template"`
	Logs      []LogConfig `json:"logs"`
}

type LogConfig struct {
	Name      string   `json:"Name"`
	Namespace *string  `json:"namespace"`
	Labels    string   `json:"labels"`
	Template  *string  `json:"template"`
	Tags      []string `json:"tags"`
}

func LoadConfig() (*Config, error) {
	var filename = GlobalFlags.ConfigPath
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura del file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("errore nel parsing del JSON: %v", err)
	}

	return &config, nil
}
