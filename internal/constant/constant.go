package constant

import (
	"serenibase/internal/dto"
	"serenibase/internal/utils/helpers"
)

const (
	MasterDatabase = "public"

	// Email footer notice
	EmailFooterNotice = `────────────────────────────────────
CONFIDENTIALITY & SECURITY NOTICE

This email and any attachments are intended solely for the designated recipient and may contain confidential or proprietary information. Unauthorized use, disclosure, or distribution is strictly prohibited.

Serenibase will never request passwords or sensitive credentials via email. Please do not share OTPs, access links, or authentication details.

This is an automated message. Replies are not monitored.

© Serenibase. All rights reserved.
────────────────────────────────────`

	// Database type constants
	DBTypeVarchar255       = "VARCHAR(255)"
	DBTypeNVarchar255      = "NVARCHAR(255)"
	DBTypeNVarcharMax      = "NVARCHAR(MAX)"
	DBTypeVarchar255Lower  = "varchar(255)"
	DBTypeVarchar100       = "varchar(100)"
	DBTypeVarchar50        = "varchar(50)"
	DBTypeVarchar150       = "varchar(150)" // Assuming for display_name
	DBTypeOracleVarchar255 = "VARCHAR2(255)"

	// Table format constants
	ModelsTableFormat = "\"%s\".models"
)

func strPtr(s string) *string {
	return &s
}

// ========== RBAC Scope Levels ==========
var ScopeLevels = struct {
	System    string
	Workspace string
	Base      string
}{
	System:    "system",
	Workspace: "workspace",
	Base:      "base",
}

// ========== RBAC Role Names ==========
var RBACRoleNames = struct {
	Owner                 string
	CoOwner               string
	WorkspaceMaintainer   string
	WorkspaceMaintainerRO string
	BaseMember            string
	BaseMemberReadOnly    string
	NoAccess              string
}{
	Owner:                 "owner",
	CoOwner:               "co-owner",
	WorkspaceMaintainer:   "maintainer",
	WorkspaceMaintainerRO: "workspace-read",
	BaseMember:            "base-member",
	BaseMemberReadOnly:    "base-read",
	NoAccess:              "user",
}

// ========== RBAC Resource Codes ==========
var ResourceCodes = struct {
	Workspace   string
	Base        string
	Table       string
	Records     string
	Members     string
	Views       string
	Settings    string
	ApiTokens   string
	Webhooks    string
	Automations string
}{
	Workspace:   "workspace",
	Base:        "base",
	Table:       "table",
	Records:     "records",
	Members:     "members",
	Views:       "views",
	Settings:    "settings",
	ApiTokens:   "api_tokens",
	Webhooks:    "webhooks",
	Automations: "automations",
}

// ========== RBAC Action Codes ==========
var ActionCodes = struct {
	Read    string
	Create  string
	Update  string
	Delete  string
	Share   string
	Invite  string
	Export  string
	Import  string
	Execute string
	Manage  string
}{
	Read:    "read",
	Create:  "create",
	Update:  "update",
	Delete:  "delete",
	Share:   "share",
	Invite:  "invite",
	Export:  "export",
	Import:  "import",
	Execute: "execute",
	Manage:  "manage",
}

// ========== Default RBAC Roles ==========
// These roles are created by default and mapped to permissions
var DefaultAccessRoles = []dto.AccessRoleDTO{
	{
		Name:        RBACRoleNames.Owner,
		ScopeLevel:  ScopeLevels.System,
		Priority:    100,
		Description: strPtr("Workspace owner with full control and management capabilities"),
		IsDefault:   false,
	},
	{
		Name:        RBACRoleNames.CoOwner,
		ScopeLevel:  ScopeLevels.System,
		Priority:    90,
		Description: strPtr("Co-owner with full access similar to owner"),
		IsDefault:   false,
	},
	{
		Name:        RBACRoleNames.WorkspaceMaintainer,
		ScopeLevel:  ScopeLevels.Workspace,
		Priority:    80,
		Description: strPtr("Workspace maintainer with elevated permissions to manage workspace"),
		IsDefault:   false,
	},
	{
		Name:        RBACRoleNames.WorkspaceMaintainerRO,
		ScopeLevel:  ScopeLevels.Workspace,
		Priority:    70,
		Description: strPtr("Workspace maintainer with read-only access"),
		IsDefault:   false,
	},
	{
		Name:        RBACRoleNames.BaseMember,
		ScopeLevel:  ScopeLevels.Base,
		Priority:    60,
		Description: strPtr("Base member with standard read and write permissions"),
		IsDefault:   false,
	},
	{
		Name:        RBACRoleNames.BaseMemberReadOnly,
		ScopeLevel:  ScopeLevels.Base,
		Priority:    50,
		Description: strPtr("Base member with read-only access"),
		IsDefault:   false,
	},
	{
		Name:        RBACRoleNames.NoAccess,
		ScopeLevel:  ScopeLevels.System,
		Priority:    10,
		Description: strPtr("No workspace access - can only view and edit own profile"),
		IsDefault:   true,
	},
}

var SystemColumns = []dto.AddColumnRequest{
	{
		Title:       "Id",
		Description: "",
		UIDT:        "number",
		DT:          "BIGSERIAL",
		OrderIndex:  helpers.Float64Ptr(0),
		Virtual:     helpers.BoolPtr(false),
		System:      helpers.BoolPtr(true),
	},
	{
		Title:       "Title",
		Description: "",
		UIDT:        "text",
		DT:          "TEXT",
		OrderIndex:  helpers.Float64Ptr(1),
		Virtual:     helpers.BoolPtr(false),
		System:      helpers.BoolPtr(false),
	},
	{
		Title:       "Created Time",
		Description: "",
		UIDT:        "datetime",
		DT:          "TIMESTAMP",
		OrderIndex:  helpers.Float64Ptr(2),
		Virtual:     helpers.BoolPtr(false),
		System:      helpers.BoolPtr(true),
	},
	{
		Title:       "Last Modified Time",
		Description: "",
		UIDT:        "datetime",
		DT:          "TIMESTAMP",
		OrderIndex:  helpers.Float64Ptr(3),
		Virtual:     helpers.BoolPtr(false),
		System:      helpers.BoolPtr(true),
	},
	{
		Title:       "Created By",
		Description: "",
		UIDT:        "createdBy",
		DT:          "TEXT",
		OrderIndex:  helpers.Float64Ptr(4),
		Virtual:     helpers.BoolPtr(false),
		System:      helpers.BoolPtr(true),
	},
	{
		Title:       "Last Modified By",
		Description: "",
		UIDT:        "lastModifiedBy",
		DT:          "TEXT",
		OrderIndex:  helpers.Float64Ptr(5),
		Virtual:     helpers.BoolPtr(false),
		System:      helpers.BoolPtr(true),
	},
}

// var SystemColumns = []dbModels.ColumnDefinition{
// 	{Name: "id", DataType: "BIGSERIAL", NotNull: true},
// 	{Name: "title", DataType: "TEXT"},
// 	{Name: "created_time", DataType: "TIMESTAMP", NotNull: true},
// 	{Name: "last_modified_time", DataType: "TIMESTAMP", NotNull: true},
// }

type DBMapping struct {
	Component string
	Label     string
	Postgres  string
	MongoDB   string
	MySQL     string
	SQLite    string
	MSSQL     string
	Oracle    string
}

// Helper functions to create common DBMapping patterns
func createTextMapping(component, label string) DBMapping {
	return DBMapping{
		Component: component,
		Label:     label,
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     DBTypeVarchar255,
		SQLite:    "TEXT",
		MSSQL:     DBTypeNVarchar255,
		Oracle:    DBTypeOracleVarchar255,
	}
}

func createNumericMapping(component, label, postgres, mysql, mssql, oracle string) DBMapping {
	return DBMapping{
		Component: component,
		Label:     label,
		Postgres:  postgres,
		MongoDB:   "Number",
		MySQL:     mysql,
		SQLite:    "REAL",
		MSSQL:     mssql,
		Oracle:    oracle,
	}
}

func createIntMapping(component, label, postgres string) DBMapping {
	return DBMapping{
		Component: component,
		Label:     label,
		Postgres:  postgres,
		MongoDB:   "Number",
		MySQL:     "INT",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER",
	}
}

func createTimestampMapping(component, label string) DBMapping {
	return DBMapping{
		Component: component,
		Label:     label,
		Postgres:  "TIMESTAMP",
		MongoDB:   "Date",
		MySQL:     "DATETIME",
		SQLite:    "TEXT",
		MSSQL:     "DATETIME2",
		Oracle:    "TIMESTAMP",
	}
}

func createArrayMapping(component, label, postgres string) DBMapping {
	return DBMapping{
		Component: component,
		Label:     label,
		Postgres:  postgres,
		MongoDB:   "Array",
		MySQL:     "JSON",
		SQLite:    "TEXT",
		MSSQL:     DBTypeNVarcharMax,
		Oracle:    "CLOB",
	}
}

func createSystemTextMapping(label string) DBMapping {
	return DBMapping{
		Component: "SystemField",
		Label:     label,
		Postgres:  "TEXT",
		MongoDB:   "ObjectId",
		MySQL:     "INT",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER",
	}
}

func createUniformTextMapping(component, label string) DBMapping {
	return DBMapping{
		Component: component,
		Label:     label,
		Postgres:  "TEXT",
		MongoDB:   "TEXT",
		MySQL:     "TEXT",
		SQLite:    "TEXT",
		MSSQL:     "TEXT",
		Oracle:    "TEXT",
	}
}

var UITypeMappings = map[string]DBMapping{
	"text": createTextMapping("TextInput", "Single Line Text"),
	"longText": {
		Component: "TextArea",
		Label:     "Long Text",
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     "TEXT",
		SQLite:    "TEXT",
		MSSQL:     "NTEXT",
		Oracle:    "CLOB",
	},
	"number":  createIntMapping("NumberInput", "Number", "INTEGER"),
	"decimal": createNumericMapping("NumberInput", "Decimal", "NUMERIC", "DECIMAL", "DECIMAL", "NUMBER"),
	"boolean": {
		Component: "Checkbox",
		Label:     "Checkbox",
		Postgres:  "BOOLEAN",
		MongoDB:   "Boolean",
		MySQL:     "TINYINT(1)",
		SQLite:    "BOOLEAN",
		MSSQL:     "BIT",
		Oracle:    "NUMBER(1)",
	},
	"currency": createNumericMapping("CurrencyInput", "Currency", "NUMERIC", "DECIMAL", "DECIMAL", "NUMBER"),
	"percent":  createNumericMapping("PercentInput", "Percent", "NUMERIC", "DECIMAL", "DECIMAL", "NUMBER"),
	"duration": {
		Component: "DurationPicker",
		Label:     "Duration",
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     "TIME",
		SQLite:    "TEXT",
		MSSQL:     "TEXT",
		Oracle:    "TEXT",
	},
	"year": {
		Component: "YearPicker",
		Label:     "Year",
		Postgres:  "INTEGER",
		MongoDB:   "Number",
		MySQL:     "YEAR",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER(4)",
	},
	"date": {
		Component: "DatePicker",
		Label:     "Date",
		Postgres:  "DATE",
		MongoDB:   "Date",
		MySQL:     "DATE",
		SQLite:    "TEXT",
		MSSQL:     "DATE",
		Oracle:    "DATE",
	},
	"datetime": createTimestampMapping("DateTimePicker", "Date Time"),
	"time": {
		Component: "TimePicker",
		Label:     "Time",
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     "TIME",
		SQLite:    "TEXT",
		MSSQL:     "TIME",
		Oracle:    "DATE",
	},
	"email": createTextMapping("EmailInput", "Email"),
	"phoneNumber": {
		Component: "PhoneInput",
		Label:     "Phone Number",
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     "VARCHAR(20)",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(20)",
		Oracle:    "VARCHAR2(20)",
	},
	"url":         createTextMapping("URLInput", "URL"),
	"select":      createTextMapping("Dropdown", "Single Select"),
	"multiSelect": createArrayMapping("MultiDropdown", "Multi Select", "TEXT[]"),
	"rating": {
		Component: "RatingStars",
		Label:     "Rating",
		Postgres:  "INT",
		MongoDB:   "Number",
		MySQL:     "TINYINT",
		SQLite:    "INTEGER",
		MSSQL:     "TINYINT",
		Oracle:    "NUMBER",
	},
	"user":   createUniformTextMapping("UserPicker", "User"),
	"button": createUniformTextMapping("Button", "Button"),
	"json": {
		Component: "JSONField",
		Label:     "JSON Field",
		Postgres:  "TEXT",
		MongoDB:   "Document",
		MySQL:     "JSON",
		SQLite:    "TEXT",
		MSSQL:     DBTypeNVarcharMax,
		Oracle:    "CLOB",
	},
	"uuid": {
		Component: "UUIDField",
		Label:     "UUID",
		Postgres:  "UUID",
		MongoDB:   "String",
		MySQL:     "CHAR(36)",
		SQLite:    "TEXT",
		MSSQL:     "UNIQUEIDENTIFIER",
		Oracle:    "RAW(16)",
	},
	"links_source_one-to-one":   createIntMapping("LinkInput", "Links", "INT"),
	"links_target_one-to-one":   createIntMapping("LinkInput", "Links", "INT"),
	"links_source_has-many":     createArrayMapping("LinkInput", "Links", "INT[]"),
	"links_target_has-many":     createIntMapping("LinkInput", "Links", "INT"),
	"links_source_many-to-many": createArrayMapping("LinkInput", "Links", "INT[]"),
	"links_target_many-to-many": createArrayMapping("LinkInput", "Links", "INT[]"),
	"lookup": {
		Component: "LookupField",
		Label:     "Lookup",
		Postgres:  "ANY",
		MongoDB:   "ObjectId",
		MySQL:     "INT",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER",
	},
	"createdTime": createTimestampMapping("SystemField", "Created Time"),
	"attachment": {
		Component: "AttachmentField",
		Label:     "Attachment",
		Postgres:  "JSONB[]",
		MongoDB:   "Array",
		MySQL:     "JSON",
		SQLite:    "TEXT",
		MSSQL:     DBTypeNVarcharMax,
		Oracle:    "CLOB",
	},
	"lastModifiedTime": createTimestampMapping("SystemField", "Last Modified Time"),
	"createdBy":        createSystemTextMapping("Created By"),
	"lastModifiedBy":   createSystemTextMapping("Last Modified By"),
	"formula":          createUniformTextMapping("Formula", "Formula"),
}

// AllowedConversions says: fromType -> list of allowed target types.
//
// Implicitly, conversion to the same type is always allowed and not repeated here.
var AllowedConversions = map[string][]string{
	// --- Text family ---
	"text": {
		"longText",
		"number", "decimal", "currency", "percent", "rating", "year",
		"date", "datetime", "time",
		"email", "phoneNumber", "url",
		"select", "multiSelect",
		"boolean",
		"json",
		"uuid",
		"duration",
	},
	"longText": {
		"text",
		"json", // if JSON-parsable
	},

	// --- Numeric-ish family ---
	"number": {
		"text", "longText",
		"decimal", "currency", "percent",
		"rating",
		"year",
	},
	"decimal": {
		"text", "longText",
		"number", // if integer-valued
		"currency", "percent",
	},
	"currency": {
		"text", "longText",
		"decimal", "number", "percent",
	},
	"percent": {
		"text", "longText",
		"decimal", "number", "currency",
	},
	"rating": {
		"text", "longText",
		"number", "decimal",
	},
	"boolean": {
		"text", "longText",
		"number", // 0/1
		"select", // "true"/"false"
	},

	// --- Date / time family ---
	"year": {
		"text", "longText",
		"number",
		"date", "datetime",
	},
	"date": {
		"text", "longText",
		"datetime",
		"year",
	},
	"datetime": {
		"text", "longText",
		"date",
		"time",
	},
	"time": {
		"text", "longText",
		"duration", // if you treat it as duration-of-day
	},
	"duration": {
		"text", "longText",
	},

	// --- Specialized text-y fields ---
	"email": {
		"text", "longText",
	},
	"phoneNumber": {
		"text", "longText",
	},
	"url": {
		"text", "longText",
	},

	// --- Select / multi-select ---
	"select": {
		"text", "longText",
	},
	"multiSelect": {
		"text", "longText",
		"json", // array serialized as JSON
	},

	// --- JSON / UUID / lookup-ish ---
	"json": {
		"text", "longText",
	},
	"uuid": {
		"text", "longText",
	},
	"user": {
		"text", "longText", // store the user id / name as plain text
	},
	"lookup": {
		"text", "longText", // store referenced id as text
	},

	// --- System fields (usually locked; you can keep them non-convertible) ---
	// "createdTime", "lastModifiedTime", "createdBy", "lastModifiedBy"
	// I'd recommend NOT allowing conversion at all, but if you want:
	"createdTime": {
		"text", "longText", "datetime",
	},
	"lastModifiedTime": {
		"text", "longText", "datetime",
	},
	"createdBy": {
		"text", "longText",
	},
	"lastModifiedBy": {
		"text", "longText",
	},

	// --- Non-data / UI / relation types: no conversions (see next section) ---
	// "button", "formula",
	// "links_*",
	// "attachment",
}
