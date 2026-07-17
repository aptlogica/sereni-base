// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	aiServiceURL string
	httpClient   *http.Client
}

func NewChatHandler(aiServiceURL string) *ChatHandler {
	return &ChatHandler{
		aiServiceURL: strings.TrimRight(aiServiceURL, "/"),
		httpClient:   &http.Client{},
	}
}

func (h *ChatHandler) ProxyChat(c *gin.Context) {
	if h.aiServiceURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CHAT_SERVICE_NOT_CONFIGURED",
				"message": "AI service URL is not configured",
			},
		})
		return
	}

	// Re-read the request body so the upstream service gets the exact payload.
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_PAYLOAD",
				"message": "Invalid JSON body",
			},
		})
		return
	}
	upstreamURL := h.aiServiceURL + "/chat"
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CHAT_PROXY_ERROR",
				"message": "Failed to create chat request",
			},
		})
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CHAT_PROXY_ERROR",
				"message": "Chat service request failed",
				"details": err.Error(),
			},
		})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CHAT_PROXY_ERROR",
				"message": "Failed to read chat service response",
				"details": err.Error(),
			},
		})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, respBody)
}
