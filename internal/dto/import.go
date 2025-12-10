package dto

type ImportTableRequest struct {
	BaseID      string `form:"base_id" json:"base_id" binding:"required"`
	WorkspaceID string `form:"workspace_id" json:"workspace_id" binding:"required"`
	TableName   string `form:"table_name" json:"table_name" binding:"required"`
	OrderIndex  int    `form:"order_index" json:"order_index"`
	CreatedBy   string `form:"created_by" json:"created_by,omitempty"`
}

type ImportTableResponse struct {
	TableResponse
}
