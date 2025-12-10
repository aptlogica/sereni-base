package constant

import (
	"serenibase/internal/dto"
	"serenibase/internal/utils/helpers"
	"strings"
)

const (
	MasterDatabase = "master"
)

func strPtr(s string) *string {
	return &s
}

var RoleNames = struct {
	Admin string
	User  string
}{
	Admin: "Admin",
	User:  "User",
}

var AccessNames = struct {
	FullAccess    string
	LimitedAccess string
}{
	FullAccess:    "full_access",
	LimitedAccess: "limited_access",
}

var DefaultRoles = []dto.RoleInsertion{
	{
		Name:        RoleNames.Admin,
		Description: strPtr("Has full administrative privileges, including managing users, system settings, billings, and subscriptions."),
		IsDefault:   false,
	},
	{
		Name:        RoleNames.User,
		Description: strPtr("Standard user with no access unless or until added to a workspace."),
		IsDefault:   false,
	},
}

var DefaultAccessLevels = []dto.RoleInsertion{
	{
		Name:        AccessNames.FullAccess,
		Description: strPtr("Administrator of the workspace with elevated permissions specific to the workspace, can also invite other members who have already been added by the admin."),
		IsDefault:   false,
	},
	{
		Name:        AccessNames.LimitedAccess,
		Description: strPtr("Member of the workspace with standard permissions."),
		IsDefault:   false,
	},
}

var PlanNames = struct {
	Free    string
	Premium string
}{
	Free:    "Free",
	Premium: "Premium",
}


var DefaultPlans = []dto.PlanInsertion{
	{
		Name:                 PlanNames.Free,
		Slug:                 strings.ToLower(PlanNames.Free),
		Description:          strPtr("Free plan with limited features"),
		Currency:             "USD",
		MaxWorkspaces:        func() *int { v := 1; return &v }(),
		MaxBasesPerWorkspace: func() *int { v := 2; return &v }(),
		MaxTablesPerBase:     func() *int { v := 5; return &v }(),
		MaxRowsPerTable:      func() *int { v := 1000; return &v }(),
		MaxCollaborators:     func() *int { v := 3; return &v }(),
		MaxAPICallsPerHour:   func() *int { v := 100; return &v }(),
		StorageLimitGB:       func() *int { v := 1; return &v }(),
		Features:             "[]",
		IsActive:             true,
	},
	{
		Name:                 PlanNames.Premium,
		Slug:                 strings.ToLower(PlanNames.Premium),
		Description:          strPtr("Premium plan with advanced features"),
		Currency:             "USD",
		MaxWorkspaces:        func() *int { v := 10; return &v }(),
		MaxBasesPerWorkspace: func() *int { v := 20; return &v }(),
		MaxTablesPerBase:     func() *int { v := 50; return &v }(),
		MaxRowsPerTable:      func() *int { v := 100000; return &v }(),
		MaxCollaborators:     func() *int { v := 50; return &v }(),
		MaxAPICallsPerHour:   func() *int { v := 10000; return &v }(),
		StorageLimitGB:       func() *int { v := 100; return &v }(),
		Features:             "[\"priority_support\",\"advanced_analytics\"]",
		IsActive:             true,
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

var UITypeMappings = map[string]DBMapping{
	"text": {
		Component: "TextInput",
		Label:     "Single Line Text",
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     "VARCHAR(255)",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(255)",
		Oracle:    "VARCHAR2(255)",
	},
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
	"number": {
		Component: "NumberInput",
		Label:     "Number",
		Postgres:  "INTEGER",
		MongoDB:   "Number",
		MySQL:     "INT",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER",
	},
	"decimal": {
		Component: "NumberInput",
		Label:     "Decimal",
		Postgres:  "NUMERIC",
		MongoDB:   "Number",
		MySQL:     "DECIMAL",
		SQLite:    "REAL",
		MSSQL:     "DECIMAL",
		Oracle:    "NUMBER",
	},
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
	"currency": {
		Component: "CurrencyInput",
		Label:     "Currency",
		Postgres:  "NUMERIC",
		MongoDB:   "Number",
		MySQL:     "DECIMAL",
		SQLite:    "REAL",
		MSSQL:     "DECIMAL",
		Oracle:    "NUMBER",
	},
	"percent": {
		Component: "PercentInput",
		Label:     "Percent",
		Postgres:  "NUMERIC",
		MongoDB:   "Number",
		MySQL:     "DECIMAL",
		SQLite:    "REAL",
		MSSQL:     "DECIMAL",
		Oracle:    "NUMBER",
	},
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
	"datetime": {
		Component: "DateTimePicker",
		Label:     "Date Time",
		Postgres:  "TIMESTAMP",
		MongoDB:   "Date",
		MySQL:     "DATETIME",
		SQLite:    "TEXT",
		MSSQL:     "DATETIME2",
		Oracle:    "TIMESTAMP",
	},
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
	"email": {
		Component: "EmailInput",
		Label:     "Email",
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     "VARCHAR(255)",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(255)",
		Oracle:    "VARCHAR2(255)",
	},
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
	"url": {
		Component: "URLInput",
		Label:     "URL",
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     "VARCHAR(255)",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(255)",
		Oracle:    "VARCHAR2(255)",
	},
	"select": {
		Component: "Dropdown",
		Label:     "Single Select",
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     "VARCHAR(255)",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(255)",
		Oracle:    "VARCHAR2(255)",
	},
	"multiSelect": {
		Component: "MultiDropdown",
		Label:     "Multi Select",
		Postgres:  "TEXT[]",
		MongoDB:   "Array",
		MySQL:     "JSON",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(MAX)",
		Oracle:    "CLOB",
	},
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
	"user": {
		Component: "UserPicker",
		Label:     "User",
		Postgres:  "TEXT",
		MongoDB:   "TEXT",
		MySQL:     "TEXT",
		SQLite:    "TEXT",
		MSSQL:     "TEXT",
		Oracle:    "TEXT",
	},
	"button": {
		Component: "Button",
		Label:     "Button",
		Postgres:  "TEXT",
		MongoDB:   "TEXT",
		MySQL:     "TEXT",
		SQLite:    "TEXT",
		MSSQL:     "TEXT",
		Oracle:    "TEXT",
	},
	"json": {
		Component: "JSONField",
		Label:     "JSON Field",
		Postgres:  "TEXT",
		MongoDB:   "Document",
		MySQL:     "JSON",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(MAX)",
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
	"links_source_one-to-one": {
		Component: "LinkInput",
		Label:     "Links",
		Postgres:  "INT",
		MongoDB:   "Number",
		MySQL:     "INT",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER",
	},
	"links_target_one-to-one": {
		Component: "LinkInput",
		Label:     "Links",
		Postgres:  "INT",
		MongoDB:   "Number",
		MySQL:     "INT",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER",
	},
	"links_source_has-many": {
		Component: "LinkInput",
		Label:     "Links",
		Postgres:  "INT[]",
		MongoDB:   "Array",
		MySQL:     "JSON",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(MAX)",
		Oracle:    "CLOB",
	},
	"links_target_has-many": {
		Component: "LinkInput",
		Label:     "Links",
		Postgres:  "INT",
		MongoDB:   "Number",
		MySQL:     "INT",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER",
	},
	"links_source_many-to-many": {
		Component: "LinkInput",
		Label:     "Links",
		Postgres:  "INT[]",
		MongoDB:   "Array",
		MySQL:     "JSON",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(MAX)",
		Oracle:    "CLOB",
	},
	"links_target_many-to-many": {
		Component: "LinkInput",
		Label:     "Links",
		Postgres:  "INT[]",
		MongoDB:   "Array",
		MySQL:     "JSON",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(MAX)",
		Oracle:    "CLOB",
	},
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
	"createdTime": {
		Component: "SystemField",
		Label:     "Created Time",
		Postgres:  "TIMESTAMP",
		MongoDB:   "Date",
		MySQL:     "DATETIME",
		SQLite:    "TEXT",
		MSSQL:     "DATETIME2",
		Oracle:    "TIMESTAMP",
	},
	"attachment": {
		Component: "AttachmentField",
		Label:     "Attachment",
		Postgres:  "JSONB[]",
		MongoDB:   "Array",
		MySQL:     "JSON",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(MAX)",
		Oracle:    "CLOB",
	},
	"lastModifiedTime": {
		Component: "SystemField",
		Label:     "Last Modified Time",
		Postgres:  "TIMESTAMP",
		MongoDB:   "Date",
		MySQL:     "DATETIME",
		SQLite:    "TEXT",
		MSSQL:     "DATETIME2",
		Oracle:    "TIMESTAMP",
	},
	"createdBy": {
		Component: "SystemField",
		Label:     "Created By",
		Postgres:  "TEXT",
		MongoDB:   "ObjectId",
		MySQL:     "INT",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER",
	},
	"lastModifiedBy": {
		Component: "SystemField",
		Label:     "Last Modified By",
		Postgres:  "TEXT",
		MongoDB:   "ObjectId",
		MySQL:     "INT",
		SQLite:    "INTEGER",
		MSSQL:     "INT",
		Oracle:    "NUMBER",
	},
	"formula": {
		Component: "Formula",
		Label:     "Formula",
		Postgres:  "TEXT",
		MongoDB:   "TEXT",
		MySQL:     "TEXT",
		SQLite:    "TEXT",
		MSSQL:     "TEXT",
		Oracle:    "TEXT",
	},
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
