package main

import (
	"log"
	"serenibase/internal/app"
	"serenibase/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	application, err := app.New(cfg)
	config.AppConfig = cfg
	if err != nil {
		log.Fatal("Failed to create application:", err)
	}

	if err := application.Run(); err != nil {
		log.Fatal("Failed to run application:", err)
	}
}
