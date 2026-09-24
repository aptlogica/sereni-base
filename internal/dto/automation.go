// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

// CreateAutomationRequest is the payload for POST /automation/create
type CreateAutomationRequest struct {
	ModelID string `json:"model_id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Type    string `json:"type" binding:"required,oneof=trigger webhook function"`
	Context string `json:"context" binding:"required"`
}
