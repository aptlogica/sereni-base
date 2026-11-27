// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/aptlogica/sereni-base/internal/providers/logger"
)

// The message file is the only place message text for the API platform lives: the log messages now, the
// error messages with the error catalog. One flat map of dotted key to entry, see
// SERENI-DOCUMENTATION/api-platform-log-shipping-suggestion.md section 14.1.
//
//go:embed messages/en-US.json
var enUSMessages []byte

const (
	messageTypeError = "error"
	messageTypeLog   = "log"

	// maxParamLength caps a placeholder value: values come from the request
	maxParamLength = 60
)

// message is one entry of the message file
type message struct {
	Type    string   `json:"type"`  // "error" or "log"
	Code    string   `json:"code"`  // error: public error code
	Level   string   `json:"level"` // log: info, warn or error
	Phase   int      `json:"phase"` // log: later phase that uses the message, 0 when used now
	Message string   `json:"message"`
	Action  string   `json:"action"` // error: what to do next
	Params  []string `json:"params"`
	Note    string   `json:"note"` // developer detail, never in a response or a log line
}

type messageFile struct {
	Locale   string             `json:"locale"`
	Version  int                `json:"version"`
	Messages map[string]message `json:"messages"`
}

// messageCatalog is a loaded message file
type messageCatalog struct {
	locale   string
	messages map[string]message
	warned   sync.Map // keys already reported missing, so a bad key warns once
}

var placeholderPattern = regexp.MustCompile(`\{([a-z_][a-z0-9_]*)\}`)

// defaultMessages is the embedded en-US file. It is checked by tests; a broken file stops the server at start.
var defaultMessages = mustLoadMessages(enUSMessages)

func mustLoadMessages(data []byte) *messageCatalog {
	c, err := loadMessages(data)
	if err != nil {
		panic("platform: invalid message file: " + err.Error())
	}
	return c
}

// loadMessages parses and checks a message file: known types, log levels, and the declared params agree
// with the {placeholders} in the text.
func loadMessages(data []byte) (*messageCatalog, error) {
	var f messageFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Locale == "" || len(f.Messages) == 0 {
		return nil, fmt.Errorf("locale or messages missing")
	}
	for key, m := range f.Messages {
		if m.Message == "" {
			return nil, fmt.Errorf("%s: message is empty", key)
		}
		switch m.Type {
		case messageTypeError:
			if m.Code == "" {
				return nil, fmt.Errorf("%s: error message without code", key)
			}
		case messageTypeLog:
			if m.Level != "info" && m.Level != "warn" && m.Level != "error" {
				return nil, fmt.Errorf("%s: log level must be info, warn or error", key)
			}
		default:
			return nil, fmt.Errorf("%s: type must be error or log", key)
		}
		used := map[string]bool{}
		for _, text := range []string{m.Message, m.Action} {
			for _, match := range placeholderPattern.FindAllStringSubmatch(text, -1) {
				used[match[1]] = true
			}
		}
		declared := map[string]bool{}
		for _, p := range m.Params {
			declared[p] = true
		}
		if !sameSet(used, declared) {
			return nil, fmt.Errorf("%s: params %v do not match the placeholders %v", key, m.Params, keys(used))
		}
	}
	return &messageCatalog{locale: f.Locale, messages: f.Messages}, nil
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// get returns the entry for key
func (c *messageCatalog) get(key string) (message, bool) {
	m, ok := c.messages[key]
	return m, ok
}

// has tells whether key is in the file
func (c *messageCatalog) has(key string) bool {
	_, ok := c.messages[key]
	return ok
}

// render fills the placeholders of key's message. A missing key renders as the key itself and is logged
// once as a warning; it never panics. A placeholder without a value stays as written.
func (c *messageCatalog) render(key string, params map[string]any) string {
	m, ok := c.messages[key]
	if !ok {
		if _, seen := c.warned.LoadOrStore(key, true); !seen {
			logger.Get().Warn().Str("key", key).Str("locale", c.locale).Msg("message key not found in the message file")
		}
		return key
	}
	return fillPlaceholders(m.Message, params)
}

// renderAction fills the placeholders of key's action, the general "what to do next" that goes with the
// message. It is empty when the key or its action is missing; a missing key is not warned about, because the
// action is an extra and the caller already rendered the message.
func (c *messageCatalog) renderAction(key string, params map[string]any) string {
	m, ok := c.messages[key]
	if !ok || m.Action == "" {
		return ""
	}
	return fillPlaceholders(m.Action, params)
}

func fillPlaceholders(text string, params map[string]any) string {
	return placeholderPattern.ReplaceAllStringFunc(text, func(ph string) string {
		v, ok := params[ph[1:len(ph)-1]]
		if !ok {
			return ph
		}
		return cleanParam(fmt.Sprint(v))
	})
}

// cleanParam strips control characters and caps the length, because placeholder values come from the request
func cleanParam(s string) string {
	return cleanText(s, maxParamLength)
}

// cleanText strips control characters and caps the text at max characters
func cleanText(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "..."
	}
	return s
}
