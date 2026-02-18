// serenibase/internal/dto/column_dto.go
package dto

import (
	"serenibase/internal/utils/helpers"
	"time"

	"github.com/google/uuid"
)

// ColumnInsertion is used when inserting a new column
type ColumnInsertion struct {
	ID      uuid.UUID `db:"id" json:"id,omitempty"`
	ModelID uuid.UUID `db:"model_id" json:"model_id,omitempty"`
	BaseID  uuid.UUID `db:"base_id" json:"base_id,omitempty"`

	// Column identification
	ColumnName  string                 `db:"column_name" json:"column_name,omitempty"`
	Title       string                 `db:"title" json:"title,omitempty"`
	Description *string                `db:"description" json:"description,omitempty" mapstructure:"description"`
	Meta        map[string]interface{} `db:"meta" json:"meta,omitempty" mapstructure:"meta"`

	// Data type information
	UIDT string  `db:"uidt" json:"uidt,omitempty"`
	DT   *string `db:"dt" json:"dt,omitempty"`

	// // Column properties
	// PK               bool `db:"pk" json:"pk,omitempty"`
	// PV               bool `db:"pv" json:"pv,omitempty"`
	// RQD              bool `db:"rqd" json:"rqd,omitempty"`
	// UN               bool `db:"un" json:"un,omitempty"`
	// AI               bool `db:"ai" json:"ai,omitempty"`
	// UniqueConstraint bool `db:"unique_constraint" json:"unique_constraint,omitempty"`

	// // Data type parameters
	// MaxLength      *int `db:"max_length" json:"max_length,omitempty"`
	// PrecisionValue *int `db:"precision_value" json:"precision_value,omitempty"`
	// ScaleValue     *int `db:"scale_value" json:"scale_value,omitempty"`

	// // Default and validation
	// DefaultValue    *string         `db:"default_value" json:"default_value,omitempty"`
	// ValidationRules json.RawMessage `db:"validation_rules" json:"validation_rules,omitempty"` // JSON

	// Special column types
	Virtual bool `db:"virtual" json:"virtual,omitempty"`
	System  bool `db:"system" json:"system,omitempty"`
	Deleted bool `db:"deleted" json:"deleted,omitempty"`

	// Display
	OrderIndex *float64  `db:"order_index" json:"order_index,omitempty"`
	CreatedBy  string    `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy  string    `db:"last_modified_by" json:"last_modified_by,omitempty"`
	CreatedAt  time.Time `db:"created_time" json:"created_time,omitempty"`
	UpdatedAt  time.Time `db:"last_modified_time" json:"last_modified_time,omitempty"`
}

// Map converts ColumnInsertion → map[string]interface{} for DB insert
func (c *ColumnInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 c.ID,
		"model_id":           c.ModelID,
		"base_id":            c.BaseID,
		"column_name":        c.ColumnName,
		"title":              c.Title,
		"uidt":               c.UIDT,
		"dt":                 c.DT,
		"description":        c.Description,
		"meta":               helpers.InterfaceToJSONString(c.Meta),
		"virtual":            c.Virtual,
		"system":             c.System,
		"deleted":            c.Deleted,
		"order_index":        c.OrderIndex,
		"created_by":         c.CreatedBy,
		"last_modified_by":   c.UpdatedBy,
		"created_time":       c.CreatedAt,
		"last_modified_time": c.UpdatedAt,
	}
}

// ColumnUpdate is used when updating an existing column
type ColumnUpdate struct {
	Title       *string                 `db:"title" json:"title,omitempty"`
	ColumnName  *string                 `db:"column_name" json:"column_name,omitempty"`
	Description *string                 `db:"description" json:"description,omitempty"`
	Meta        *map[string]interface{} `db:"meta" json:"meta,omitempty"`
	UIDT        *string                 `db:"uidt" json:"uidt,omitempty"`
	DT          *string                 `db:"dt" json:"dt,omitempty"`
	Virtual     *bool                   `db:"virtual" json:"virtual,omitempty"`
	System      *bool                   `db:"system" json:"system,omitempty"`
	Deleted     *bool                   `db:"deleted" json:"deleted,omitempty"`
	OrderIndex  *float64                `db:"order_index" json:"order_index,omitempty"`
	UpdatedBy   string                  `db:"last_modified_by" json:"last_modified_by,omitempty"`
	UpdatedAt   time.Time               `db:"last_modified_time" json:"last_modified_time,omitempty"`
}

// Map converts ColumnUpdate → map[string]interface{} for DB update
func (c *ColumnUpdate) Map() map[string]interface{} {
	result := make(map[string]interface{})
	if c.Title != nil {
		result["title"] = *c.Title
	}
	if c.ColumnName != nil {
		result["column_name"] = *c.ColumnName
	}
	if c.Description != nil {
		result["description"] = *c.Description
	}
	if c.Meta != nil {
		result["meta"] = helpers.InterfaceToJSONString(*c.Meta)
	}
	if c.DT != nil {
		result["dt"] = *c.DT
	}

	if c.UIDT != nil {
		result["uidt"] = *c.UIDT
	}
	// if c.PK != nil {
	// 	result["pk"] = *c.PK
	// }
	// if c.PV != nil {
	// 	result["pv"] = *c.PV
	// }
	// if c.RQD != nil {
	// 	result["rqd"] = *c.RQD
	// }
	// if c.UN != nil {
	// 	result["un"] = *c.UN
	// }
	// if c.AI != nil {
	// 	result["ai"] = *c.AI
	// }
	// if c.UniqueConstraint != nil {
	// 	result["unique_constraint"] = *c.UniqueConstraint
	// }
	// if c.MaxLength != nil {
	// 	result["max_length"] = *c.MaxLength
	// }
	// if c.PrecisionValue != nil {
	// 	result["precision_value"] = *c.PrecisionValue
	// }
	// if c.ScaleValue != nil {
	// 	result["scale_value"] = *c.ScaleValue
	// }
	// if c.DefaultValue != nil {
	// 	result["default_value"] = *c.DefaultValue
	// }
	// if c.ValidationRules != nil {
	// 	result["validation_rules"] = *c.ValidationRules
	// }
	if c.Virtual != nil {
		result["virtual"] = *c.Virtual
	}
	if c.System != nil {
		result["system"] = *c.System
	}
	if c.Deleted != nil {
		result["deleted"] = *c.Deleted
	}
	if c.OrderIndex != nil {
		result["order_index"] = *c.OrderIndex
	}
	if c.UpdatedBy != "" {
		result["last_modified_by"] = c.UpdatedBy
	}
	result["last_modified_time"] = c.UpdatedAt
	return result
}

// CreateColumnRequest is a lighter struct for API request payload
type AddColumnRequest struct {
	ModelID     uuid.UUID              `json:"model_id" mapstructure:"model_id"`
	BaseID      uuid.UUID              `json:"base_id" mapstructure:"base_id"`
	Title       string                 `json:"title" mapstructure:"title"`
	Description string                 `json:"description,omitempty" mapstructure:"description,omitempty"`
	Meta        map[string]interface{} `json:"meta" mapstructure:"meta"`
	UIDT        string                 `json:"uidt" mapstructure:"uidt"`
	DT          string                 `json:"dt" mapstructure:"dt"`
	OrderIndex  *float64               `json:"order_index,omitempty" mapstructure:"order_index,omitempty"`
	Virtual     *bool                  `json:"virtual,omitempty" mapstructure:"virtual,omitempty"`
	System      *bool                  `json:"system,omitempty" mapstructure:"system,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
}

// need to implement
// type ColumnMeta struct {
// 	Relation
// }

type ColumnResponse struct {
	ID          uuid.UUID              `json:"id" mapstructure:"id"`
	ModelID     uuid.UUID              `json:"model_id" mapstructure:"model_id"`
	BaseID      uuid.UUID              `json:"base_id" mapstructure:"base_id"`
	ColumnName  string                 `json:"column_name" mapstructure:"column_name"`
	Title       string                 `json:"title" mapstructure:"title"`
	UIDT        string                 `json:"uidt" mapstructure:"uidt"`
	DT          string                 `json:"dt" mapstructure:"dt"`
	Description string                 `json:"description" mapstructure:"description"`
	Meta        map[string]interface{} `json:"meta" mapstructure:"meta"`
	Virtual     *bool                  `json:"virtual" mapstructure:"virtual"`
	System      *bool                  `json:"system" mapstructure:"system"`
	Deleted     *bool                  `json:"deleted" mapstructure:"deleted"`
	OrderIndex  *float64               `json:"order_index" mapstructure:"order_index"`
	CreatedBy   string                 `json:"created_by" mapstructure:"created_by"`
	UpdatedBy   string                 `json:"last_modified_by" mapstructure:"last_modified_by"`
	CreatedAt   time.Time              `json:"created_time" mapstructure:"created_time"`
	UpdatedAt   time.Time              `json:"last_modified_time" mapstructure:"last_modified_time"`
}

type ReorderColumnRequest struct {
	SourceColumnID uuid.UUID `json:"source_column_id,omitempty"`
	TargetColumnID uuid.UUID `json:"target_column_id,omitempty"`
}
