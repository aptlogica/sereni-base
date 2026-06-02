package main

import (
	"encoding/json"
	// "fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/openai/openai-go"
)

// newExtractSchemaHandler returns an HTTP handler that accepts a JSON body
// with a "prompt" field, calls the OpenAI API, and returns the extracted schema.
func newExtractSchemaHandler(client *openai.Client, systemPrompt string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		var req ExtractSchemaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		userInput := strings.TrimSpace(req.Prompt)
		if userInput == "" {
			http.Error(w, "prompt is required", http.StatusBadRequest)
			return
		}

		response, err := extractSchema(client, systemPrompt, userInput)
		if err != nil {
			log.Printf("extractSchema error: %v", err)
			http.Error(w, "failed to extract schema", http.StatusInternalServerError)
			return
		}

		// Try to interpret response as JSON and return it directly.
		// If it's not valid JSON, fall back to wrapping it in a JSON object.
		var parsed any
		var responseJSON []byte
		if err := json.Unmarshal([]byte(response), &parsed); err == nil {
			responseJSON, _ = json.MarshalIndent(parsed, "", "  ")
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(parsed); err != nil {
				log.Printf("encode parsed JSON error: %v", err)
			}
		} else {
			// Fallback: send as { "result": "<string>" }
			responseJSON, _ = json.MarshalIndent(ExtractSchemaAPIResponse{Result: response}, "", "  ")
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(ExtractSchemaAPIResponse{Result: response}); err != nil {
				log.Printf("encode response error: %v", err)
			}
		}

		// Save response JSON to file
		if err := saveResponseJSON(responseJSON); err != nil {
			log.Printf("failed to save response JSON: %v", err)
		}
	}
}

// func newExtractCSVHandler(client *openai.Client, systemPrompt string) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != http.MethodPost {
// 			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
// 			return
// 		}

// 		defer r.Body.Close()

// 		var req ExtractSchemaRequest
// 		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 			http.Error(w, "invalid JSON body", http.StatusBadRequest)
// 			return
// 		}

// 		userInput := strings.TrimSpace(req.Prompt)
// 		if userInput == "" {
// 			// Allow passing columns as an array instead of a free-form prompt
// 			if len(req.Columns) == 0 {
// 				http.Error(w, "either 'prompt' or 'Columns' is required", http.StatusBadRequest)
// 				return
// 			}
// 			userInput = "Columns: " + strings.Join(req.Columns, ", ")
// 			// If types provided, append them and a paired representation when lengths match
// 			if len(req.Types) > 0 {
// 				userInput += "\nTypes: " + strings.Join(req.Types, ", ")
// 				if len(req.Types) == len(req.Columns) {
// 					pairs := make([]string, 0, len(req.Columns))
// 					for i := range req.Columns {
// 						pairs = append(pairs, req.Columns[i]+":"+req.Types[i])
// 					}
// 					userInput += "\nColumnsWithTypes: " + strings.Join(pairs, ", ")
// 				}
// 			}
// 			// fmt.Println("user input----->", userInput)
// 		}

// 		// Replace the {{ROW_COUNT}} placeholder in the system prompt.
// 		rowCount := req.RowCount
// 		if rowCount <= 0 {
// 			rowCount = 10
// 		}
// 		promptWithCount := strings.ReplaceAll(systemPrompt, "{{ROW_COUNT}}", strconv.Itoa(rowCount))
// 		fmt.Println("promptWithCount----->", promptWithCount)

// 		response, err := extractCSVOutput(client, promptWithCount, userInput)
// 		// fmt.Println("response----->", response)
// 		if err != nil {
// 			log.Printf("extractCSVOutput error: %v", err)
// 			http.Error(w, "failed to extract CSV", http.StatusInternalServerError)
// 			return
// 		}

// 		w.Header().Set("Content-Type", "text/csv")
// 		if _, err := w.Write([]byte(response)); err != nil {
// 			log.Printf("write CSV response error: %v", err)
// 		}

// 		if err := saveResponseCSV(response); err != nil {
// 			log.Printf("failed to save CSV response: %v", err)
// 		}
// 	}
// }

func newExtractCSVHandler(client *openai.Client, systemPrompt string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		var req ExtractSchemaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		userInput := strings.TrimSpace(req.Prompt)
		if userInput == "" {
			userInput = "generate csv"
		}

		rowCount := req.RowCount
		if rowCount <= 0 {
			rowCount = 10
		}
		promptWithCount := strings.ReplaceAll(systemPrompt, "{{ROW_COUNT}}", strconv.Itoa(rowCount))

		tableJSON := ""
		if req.Table != nil {
			if b, err := json.MarshalIndent(req.Table, "", "  "); err == nil {
				tableJSON = string(b)
			}
		}
		promptWithTable := strings.ReplaceAll(promptWithCount, "{{TABLE_JSON}}", tableJSON)

		response, err := extractCSVOutput(client, promptWithTable, userInput)
		if err != nil {
			log.Printf("extractCSVOutput error: %v", err)
			http.Error(w, "failed to extract CSV", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/csv")
		if _, err := w.Write([]byte(response)); err != nil {
			log.Printf("write CSV response error: %v", err)
		}

		if err := saveResponseCSV(response); err != nil {
			log.Printf("failed to save CSV response: %v", err)
		}
	}
}
func newExtractBaseSchemaHandler(client *openai.Client, systemPrompt string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		var req ExtractSchemaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		userInput := strings.TrimSpace(req.Prompt)
		if userInput == "" {
			if len(req.Columns) == 0 {
				http.Error(w, "either 'prompt' or 'Columns' is required", http.StatusBadRequest)
				return
			}
			userInput = "Columns: " + strings.Join(req.Columns, ", ")
		}

		// Initial extraction: do not infer or create relationships between tables.
		// The relationships step (links/relations) should be done in a later call.
		// userInput = strings.TrimSpace("IMPORTANT: Do NOT create any relationships between tables. Do NOT add any 'links' fields. Set 'relations' to an empty array [] for every table, and do not infer related tables unless explicitly stated.\n\n" + userInput)

		response, err := extractBaseSchema(client, systemPrompt, userInput)
		if err != nil {
			log.Printf("extractBaseSchema error: %v", err)
			http.Error(w, "failed to extract base schema", http.StatusInternalServerError)
			return
		}

		var parsed any
		var responseJSON []byte
		if err := json.Unmarshal([]byte(response), &parsed); err == nil {
			responseJSON, _ = json.MarshalIndent(parsed, "", "  ")
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(parsed); err != nil {
				log.Printf("encode parsed JSON error: %v", err)
			}
		} else {
			responseJSON, _ = json.MarshalIndent(ExtractSchemaAPIResponse{Result: response}, "", "  ")
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(ExtractSchemaAPIResponse{Result: response}); err != nil {
				log.Printf("encode response error: %v", err)
			}
		}

		if err := saveBaseLevelTablesJSON(responseJSON); err != nil {
			log.Printf("failed to save base-level tables JSON: %v", err)
		}
	}
}

func saveResponseCSV(response string) error {
	dir := "response_csv"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(dir, "database_prompt_csv_"+timestamp+".csv")

	if err := os.WriteFile(filename, []byte(response), 0644); err != nil {
		return err
	}

	log.Printf("Response CSV saved to %s", filename)
	return nil
}

func saveBaseLevelTablesJSON(responseJSON []byte) error {
	dir := "base_level_tables"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(dir, "base_level_tables_"+timestamp+".json")

	if err := os.WriteFile(filename, responseJSON, 0644); err != nil {
		return err
	}

	log.Printf("Base-level tables JSON saved to %s", filename)
	return nil
}

// saveResponseJSON saves the response JSON to a file in the response_json folder.
// The filename is based on line 17 of main.go which loads "database.prompt".
func saveResponseJSON(responseJSON []byte) error {
	// Create response_json directory if it doesn't exist
	dir := "response_json"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Generate filename based on database.prompt (from main.go:17) with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(dir, "database_prompt_types_"+timestamp+".json")

	// Write JSON to file
	if err := os.WriteFile(filename, responseJSON, 0644); err != nil {
		return err
	}

	log.Printf("Response JSON saved to %s", filename)
	return nil
}
