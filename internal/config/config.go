// internal/config/config.go
package config

import (
	"fmt"
	"os"

	"text/template"
)

type Config struct {
	Port     string
	Template *template.Template
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	content, err := os.ReadFile("katana-template.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read katana template file: %w", err)
	}

	tmpl, err := template.New("katana").Parse(string(content))
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:     port,
		Template: tmpl,
	}, nil
}
