package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Initialize configuration (.env + environment)
	if err := initConfig(); err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// Read the system prompt from database_meta.prompt file
	// systemPromptMeta, err := loadSystemPrompt("prompts/relation_between_tables.prompt")
	systemPromptMeta, err := loadSystemPrompt("prompts/multiple_tables_relation.prompt")
	if err != nil {
		fmt.Println(err)
		return
	}

	systemPromptCSV, err := loadSystemPrompt("prompts/sample_data_creation.prompt")
	if err != nil {
		fmt.Println(err)
		return
	}
	//
	// systemPromptMultipleTables, err := loadSystemPrompt("prompts/create_multiple_tables.prompt")
	// systemPromptMultipleTables, err := loadSystemPrompt("prompts/relation_between_tables.prompt")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// Initialize OpenAI client using API key from configuration
	client, err := newOpenAIClientFromEnv()
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Please set it in a .env file or as a system environment variable, e.g.:")
		fmt.Println("  OPENAI_API_KEY=your-api-key   (in .env)")
		fmt.Println("or")
		fmt.Println("  export OPENAI_API_KEY='your-api-key'   (Unix)")
		fmt.Println("  setx OPENAI_API_KEY \"your-api-key\"   (Windows)")
		return
	}

	// HTTP handlers exposed under /api/v1 prefix
	http.Handle("/api/chat", withCORS(http.HandlerFunc(chatHandler)))
	http.Handle("/api/v1/chat", withCORS(http.HandlerFunc(chatHandler)))
	http.Handle("/api/v1/extract-schema", withCORS(newExtractSchemaHandler(client, systemPromptMeta)))
	http.Handle("/api/v1/extract-schema-csv", withCORS(newExtractCSVHandler(client, systemPromptCSV)))
	http.Handle("/api/v1/extract-schema-multiple", withCORS(newExtractBaseSchemaHandler(client, systemPromptMeta)))

	port := os.Getenv("SERVER_PORT")

	if port == "" {
		port = os.Getenv("PORT")
	}

	if port == "" {
		port = "8888"
	}

	addr := ":" + port
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// withCORS is a simple CORS middleware for the default HTTP mux.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Adjust allowed origin(s) as needed, e.g. "http://localhost:3000"
		w.Header().Set("Access-Control-Allow-Origin", "*, http://localhost:8080")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight (OPTIONS) requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
