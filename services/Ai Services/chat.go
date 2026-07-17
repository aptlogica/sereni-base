package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, chatResponse{Error: "Method not allowed"})
		return
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		writeJSON(w, http.StatusInternalServerError, chatResponse{Error: "Missing OPENAI_API_KEY"})
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, chatResponse{Error: "Invalid JSON body"})
		return
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	temperature := 0.7
	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 1 {
			writeJSON(w, http.StatusBadRequest, chatResponse{Error: "temperature must be between 0 and 1"})
			return
		}
		temperature = *req.Temperature
	}

	payload := openAIResponsesAPIRequest{
		Model:        model,
		Input:        formatTranscript(req.Messages),
		Instructions: buildInstructions(req.Messages),
	}
	if modelSupportsTemperature(model) {
		payload.Temperature = &temperature
	}

	body, err := json.Marshal(payload)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, chatResponse{Error: "Failed to build OpenAI request"})
		return
	}

	httpReq, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, chatResponse{Error: "Failed to create OpenAI request"})
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Printf("OpenAI transport error: %v", err)
		writeJSON(w, http.StatusBadGateway, chatResponse{Error: "OpenAI request failed", Detail: err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("OpenAI read error: %v", err)
		writeJSON(w, http.StatusBadGateway, chatResponse{Error: "Failed to read OpenAI response", Detail: err.Error()})
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("OpenAI API error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		writeJSON(w, resp.StatusCode, chatResponse{
			Error:  fmt.Sprintf("OpenAI request failed: %s", strings.TrimSpace(string(respBody))),
			Detail: fmt.Sprintf("status=%d", resp.StatusCode),
		})
		return
	}

	var apiResp openAIResponsesAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		log.Printf("OpenAI decode error: %v body=%s", err, strings.TrimSpace(string(respBody)))
		writeJSON(w, http.StatusBadGateway, chatResponse{Error: "Invalid OpenAI response", Detail: err.Error()})
		return
	}

	reply := strings.TrimSpace(apiResp.OutputText)
	if reply == "" {
		for _, item := range apiResp.Output {
			if len(item.Content) == 0 {
				continue
			}
			reply = strings.TrimSpace(item.Content[0].Text)
			if reply != "" {
				break
			}
		}
	}

	if reply == "" {
		writeJSON(w, http.StatusBadGateway, chatResponse{Error: "Empty completion from OpenAI"})
		return
	}

	writeJSON(w, http.StatusOK, chatResponse{Reply: reply})
}

// CORS is only needed because the frontend and backend run on different ports in dev.
// func corsMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Access-Control-Allow-Origin", "*")
// 		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
// 		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

// 		if r.Method == http.MethodOptions {
// 			w.WriteHeader(http.StatusNoContent)
// 			return
// 		}

// 		next.ServeHTTP(w, r)
// 	})
// }

// writeJSON keeps all API responses consistent.
func writeJSON(w http.ResponseWriter, status int, payload chatResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// formatTranscript converts the FE message list into a prompt-like text block.
func formatTranscript(messages []apiMessage) string {
	var builder strings.Builder

	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		if role == "" {
			role = "user"
		}

		builder.WriteString(strings.ToUpper(role))
		builder.WriteString(": ")
		builder.WriteString(strings.TrimSpace(message.Content))
		builder.WriteString("\n")
	}

	return builder.String()
}

func buildInstructions(messages []apiMessage) string {
	parts := []string{
		"You are BotChat, a concise helpful assistant.",
		fmt.Sprintf("The current server date is %s.", time.Now().Format("2006-01-02")),
		"If the user asks for today's date, current date, or similar time-sensitive information, answer using the server date above.",
		"Do not echo stale dates from the conversation or from earlier assistant messages.",
	}

	if hasDateQuestion(messages) {
		parts = append(parts, "This turn is date-sensitive. Be explicit and accurate.")
	}

	return strings.Join(parts, " ")
}

func hasDateQuestion(messages []apiMessage) bool {
	for _, message := range messages {
		if strings.ToLower(message.Role) != "user" {
			continue
		}

		content := strings.ToLower(message.Content)
		if strings.Contains(content, "today") ||
			strings.Contains(content, "current date") ||
			strings.Contains(content, "date") ||
			strings.Contains(content, "yesterday") ||
			strings.Contains(content, "tomorrow") ||
			strings.Contains(content, "now") {
			return true
		}
	}

	return false
}

func modelSupportsTemperature(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if strings.HasPrefix(model, "gpt-5") {
		return false
	}
	if strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") {
		return false
	}
	return true
}
