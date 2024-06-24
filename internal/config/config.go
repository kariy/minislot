// internal/config/config.go
package config

import (
	"io/ioutil"
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

	tmplContent, err := ioutil.ReadFile("katana-template.yaml")
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("katana").Parse(string(tmplContent))
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:     port,
		Template: tmpl,
	}, nil
}
