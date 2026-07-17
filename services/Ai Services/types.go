package main

// Data model types and API request/response payloads.

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Relation struct {
	Type        string `json:"type"`
	SourceTable string `json:"source_table"`
	TargetTable string `json:"target_table"`
}

type Field struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// Constraints []string `json:"constraints"`
	Meta any `json:"meta"`
}

type Table struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

type SchemaResponse struct {
	Tables    []Table    `json:"tables"`
	Relations []Relation `json:"relations"`
}

type ExtractSchemaAPIResponse struct {
	Result string `json:"result"`
}

type ExtractSchemaRequest struct {
	Prompt   string              `json:"prompt"`
	Table    map[string]any      `json:"table,omitempty"`
	RowCount int                 `json:"row_count,omitempty"`
	Columns  []string            `json:"columns,omitempty"`
	Types    []string            `json:"types,omitempty"`
	Options  map[string][]string `json:"options,omitempty"`
}

type chatRequest struct {
	Messages    []apiMessage `json:"messages"`
	Model       string       `json:"model,omitempty"`
	Temperature *float64     `json:"temperature,omitempty"`
}

type chatResponse struct {
	Reply  string `json:"reply,omitempty"`
	Error  string `json:"error,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type openAIResponsesAPIRequest struct {
	Model        string   `json:"model"`
	Input        string   `json:"input"`
	Instructions string   `json:"instructions,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
}

type openAIResponsesAPIResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}
