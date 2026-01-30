package dto

import "time"

type UserResetTokenInsertion struct {
	ID      string    `json:"id"`
	UserID  string    `json:"user_id"`
	Token   string    `json:"token"`
	IssuedAt time.Time `json:"issued_at"`
}

// Map transforms the DTO into a map suitable for creating DB records.
func (u UserResetTokenInsertion) Map() map[string]interface{} {
	   return map[string]interface{}{
		   "id":       u.ID,
		   "user_id":  u.UserID,
		   "token":    u.Token,
		   "issued_at": u.IssuedAt,
	   }
}
