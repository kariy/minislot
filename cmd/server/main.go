// cmd/server/main.go
package main

import (
	"log"

	"github.com/kariy/minislot/internal/config"
	"github.com/kariy/minislot/internal/k8s"
	"github.com/kariy/minislot/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	client, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	srv := server.New(cfg, client)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
