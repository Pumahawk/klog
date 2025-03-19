package main

import (
	"encoding/json"
	"fmt"
	yaml "gopkg.in/yaml.v3"
	"os"
	"regexp"
)

type TemplateVars = map[string]any

type Config struct {
	Namespace *string      `json:"namespace"`
	Template  *string      `json:"template"`
	Vars      TemplateVars `json:"vars"`
	Logs      []LogConfig  `json:"logs"`
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
	isYaml, _ := regexp.MatchString("\\.yaml$", filename)
	var confLoader ConfLoader
	if isYaml {
		confLoader = yaml.Unmarshal
	} else {
		confLoader = json.Unmarshal
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura del file: %v", err)
	}

	var config Config
	if err := confLoader(data, &config); err != nil {
		return nil, fmt.Errorf("errore nel parsing del JSON: %v", err)
	}

	return &config, nil
}

type ConfLoader = func(in []byte, out interface{}) (err error)
