// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

type UserResetTokenInsertion struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Token    string `json:"token"`
	IssuedAt string `json:"issued_at"`
}

// Map transforms the DTO into a map suitable for creating DB records.
func (u UserResetTokenInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":        u.ID,
		"user_id":   u.UserID,
		"token":     u.Token,
		"issued_at": u.IssuedAt,
	}
}
