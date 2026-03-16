/*

Copyright (c) 2026 Aptlogica Technologies Private Limited
This file is part of software developed by Aptlogica Technologies Private Limited.
Licensed under the MIT License. See the LICENSE file in the project root
for full license information.
Websites:
https://www.aptlogica.com
https://www.serenibase.com
Support:
support@aptlogica.com
support@serenibase.com
*/

package main

import (
	"log"
	"github.com/aptlogica/sereni-base/internal/app"
	"github.com/aptlogica/sereni-base/internal/config"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	cfg.Server.Version = version

	application, err := app.New(cfg)
	config.AppConfig = cfg
	if err != nil {
		log.Fatal("Failed to create application:", err)
	}

	if err := application.Run(); err != nil {
		log.Fatal("Failed to run application:", err)
	}
}
