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

// import (
// 	"fmt"
// 	"serenibase/internal/services"

// 	// _ "serenibase/docs"

// 	"godbgrest/pkg"
// 	dbConfig "godbgrest/pkg/config"

// 	"golang.org/x/net/context"
// )

// func main() {
// 	dbCfg, err := dbConfig.Load()

// 	// Initialize database service for repository
// 	dbService := pkg.NewDatabaseService()

// 	db, err := dbService.Connect(dbCfg)
// 	if err != nil {
// 		fmt.Printf("failed to connect to db: %v", err)
// 		return
// 	}

// 	// Initialize services
// 	if err := dbService.InitServices(db); err != nil {
// 		fmt.Printf("failed to init services: %v", err)
// 		return
// 	}

// 	// Initialize services
// 	userService := services.NewUserService(dbService)

// 	updateData := map[string]interface{}{
// 		"status":         "active",
// 		"email_verified": true,
// 	}

// 	ctx := context.Background()

// 	// Example: update user with a specific ID (replace with actual user ID)
// 	userID := "6506930f-e1fd-48ce-8fe3-676b81773feb"

// 	_, err = userService.UpdateUser(ctx, userID, updateData)
// 	if err != nil {
// 		fmt.Printf("failed to update user: %v", err)
// 	}

// }
