// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	tableServices "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/lib/pq"
)

// Field types, by lowercased uidt. The backend spells some uidt values inconsistently (longText / longtext),
// so fields are matched on the lowercased value.
const (
	typeText             = "text"
	typeLongText         = "longtext"
	typeEmail            = "email"
	typeURL              = "url"
	typePhone            = "phonenumber"
	typeNumber           = "number"
	typeDecimal          = "decimal"
	typeCurrency         = "currency"
	typePercent          = "percent"
	typeRating           = "rating"
	typeYear             = "year"
	typeDate             = "date"
	typeDateTime         = "datetime"
	typeTime             = "time"
	typeDuration         = "duration"
	typeCheckbox         = "boolean"
	typeSelect           = "select"
	typeMultiSelect      = "multiselect"
	typeJSON             = "json"
	typeUUID             = "uuid"
	typeLinks            = "links"
	typeLookup           = "lookup"
	typeAttachment       = "attachment"
	typeUser             = "user"
	typeFormula          = "formula"
	typeButton           = "button"
	typeCreatedTime      = "createdtime"
	typeLastModifiedTime = "lastmodifiedtime"
	typeCreatedBy        = "createdby"
	typeLastModifiedBy   = "lastmodifiedby"
)

// apiTypeNames is the field type the API reports for each uidt
var apiTypeNames = map[string]string{
	typeText: "text", typeLongText: "long_text", typeEmail: "email", typeURL: "url", typePhone: "phone_number",
	typeNumber: "number", typeDecimal: "decimal", typeCurrency: "currency", typePercent: "percent",
	typeRating: "rating", typeYear: "year", typeDate: "date", typeDateTime: "date_time", typeTime: "time",
	typeDuration: "duration", typeCheckbox: "checkbox", typeSelect: "single_select", typeMultiSelect: "multi_select",
	typeJSON: "json", typeUUID: "uuid", typeLinks: "links", typeLookup: "lookup", typeAttachment: "attachment",
	typeUser: "user", typeFormula: "formula", typeCreatedTime: "created_time", typeLastModifiedTime: "last_modified_time",
	typeCreatedBy: "created_by", typeLastModifiedBy: "last_modified_by",
}

// Types the API never writes: computed, set by the system, or (attachments, users) not supported yet
var readOnlyTypes = map[string]bool{
	typeLookup: true, typeFormula: true, typeAttachment: true, typeUser: true,
	typeCreatedTime: true, typeLastModifiedTime: true, typeCreatedBy: true, typeLastModifiedBy: true,
}

// System columns every user table has; they are the record's id, created_at and updated_at, never fields
var systemColumns = map[string]bool{
	"id": true, "created_time": true, "last_modified_time": true, "created_by": true, "last_modified_by": true,
}

// field is a user field of a table as the API sees it
type field struct {
	ID         string // columns.id
	Name       string // columns.title
	ColumnName string // physical column
	Type       string // lowercased uidt
	DT         string
	Order      float64
	ReadOnly   bool
	Options    []string // select and multi-select option names
	RatingMax  int
	Multiple   bool // user fields with meta.allowMultiple

	// Links: the linked table and whether the field holds several records
	LinkTargetModelID string
	LinkMulti         bool
	// Lookups: the link field in this table and the looked-up field in the linked table
	LookupLinkColumnID   string
	LookupTargetColumnID string
}

// PublicID returns fld_ + the column ID
func (f *field) PublicID() string { return EncodeUUID(PrefixField, f.ID) }

// APIType returns the field type shown to API clients
func (f *field) APIType() string {
	if name, ok := apiTypeNames[f.Type]; ok {
		return name
	}
	return f.Type
}

// physical reports whether the field has a column in the records table (lookups are computed)
func (f *field) physical() bool { return f.Type != typeLookup }

// isArray reports whether the field's value is a list, returned as [] when empty
func (f *field) isArray() bool {
	return f.Type == typeMultiSelect || f.Type == typeLinks || f.Type == typeAttachment ||
		(f.Type == typeUser && f.Multiple) || (f.Type == typeLookup)
}

// newField builds a field from a columns row; ok is false for system columns and buttons, which are not data
func newField(id, title, columnName, uidt, dt string, system bool, order float64, meta map[string]interface{}) (*field, bool) {
	if system && systemColumns[columnName] {
		return nil, false
	}
	f := &field{ID: id, Name: title, ColumnName: columnName, Type: strings.ToLower(uidt), DT: dt, Order: order}
	if f.Type == typeButton {
		return nil, false
	}
	_, known := apiTypeNames[f.Type]
	f.ReadOnly = readOnlyTypes[f.Type] || !known

	switch f.Type {
	case typeSelect, typeMultiSelect:
		f.Options = selectOptions(meta["options"])
	case typeRating:
		f.RatingMax = 5
		if n, ok := metaNumber(meta["ratingMax"]); ok && n > 0 {
			f.RatingMax = int(n)
		}
	case typeUser:
		f.Multiple, _ = meta["allowMultiple"].(bool)
	case typeUUID:
		if cfg, ok := dto.IDGenerationFromMeta(meta); ok && cfg.Mode == constant.IDModeAuto {
			f.ReadOnly = true
		}
	case typeLinks:
		role, _ := meta["entity_role"].(string)
		if relation, ok := meta["relation"].(map[string]interface{}); ok {
			f.LinkTargetModelID, _ = relation["with"].(string)
			relationType, _ := relation["type"].(string)
			f.LinkMulti = constant.LinkIsMulti(role, relationType)
		}
	case typeLookup:
		f.LookupLinkColumnID, _ = meta["link_column_id"].(string)
		f.LookupTargetColumnID, _ = meta["lookup_column_id"].(string)
	}
	return f, true
}

// selectOptions reads option names from meta.options: [{option, color}] or plain strings
func selectOptions(raw interface{}) []string {
	list, _ := raw.([]interface{})
	options := make([]string, 0, len(list))
	for _, item := range list {
		switch v := item.(type) {
		case string:
			options = append(options, v)
		case map[string]interface{}:
			if name, ok := v["option"].(string); ok {
				options = append(options, name)
			}
		}
	}
	return options
}

func metaNumber(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

// ---- Database value -> API value ----

// Layouts of timestamps as row_to_json writes them (timestamp without time zone is UTC in SereniBase)
var dbTimeLayouts = []string{"2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999"}

// apiTime turns a stored timestamp into RFC 3339 in UTC; values it can't read are returned as stored
func apiTime(v interface{}) interface{} {
	s, ok := v.(string)
	if !ok {
		return v
	}
	for _, layout := range dbTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format(time.RFC3339Nano)
		}
	}
	return s
}

// apiValue converts a stored value (decoded from row_to_json with UseNumber) to the API's JSON value.
// Links and lookups are resolved by the caller, which needs other tables.
func apiValue(f *field, v interface{}) interface{} {
	if v == nil {
		if f.isArray() {
			return []interface{}{}
		}
		return nil
	}
	switch f.Type {
	case typeNumber, typeRating, typeYear:
		if n, ok := v.(json.Number); ok {
			if i, err := n.Int64(); err == nil {
				return i
			}
		}
		return v
	case typeDuration:
		// Stored as text: the number of minutes
		s := strings.TrimSpace(fmt.Sprint(v))
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			return json.Number(s)
		}
		return nil
	case typeDateTime, typeCreatedTime, typeLastModifiedTime:
		return apiTime(v)
	case typeMultiSelect:
		return stringList(v)
	case typeJSON:
		// Stored as text; returned as the JSON it holds
		if s, ok := v.(string); ok {
			if json.Valid([]byte(s)) {
				return json.RawMessage(s)
			}
			return s
		}
		return v
	case typeUser:
		s := fmt.Sprint(v)
		if !f.Multiple {
			return s
		}
		ids := []interface{}{}
		for _, id := range strings.Split(s, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
		return ids
	case typeAttachment:
		return apiAttachments(v)
	}
	return v
}

func stringList(v interface{}) []interface{} {
	list, ok := v.([]interface{})
	if !ok {
		return []interface{}{v}
	}
	return list
}

// apiAttachments returns {id, url, name, size, type} for each stored asset
func apiAttachments(v interface{}) []interface{} {
	out := []interface{}{}
	for _, item := range stringList(v) {
		asset, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":   asset["id"],
			"url":  asset["url"],
			"name": asset["title"],
			"size": asset["size"],
			"type": asset["mime_type"],
		})
	}
	return out
}

// ---- API value -> database value ----

var (
	emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	urlPattern   = regexp.MustCompile(`^(https?://)?[^\s/$.?#][^\s]*\.[^\s]+$`)
	phonePattern = regexp.MustCompile(`^\+?[0-9 ()\-.]{3,25}$`)
	timePattern  = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

// dbValue converts a value from a request body (decoded with UseNumber) to what the field's column stores.
// issue is a short reason when the value is not valid for the field. Links are handled by the caller.
func dbValue(f *field, v interface{}) (value interface{}, issue string) {
	if f.ReadOnly {
		return nil, "is read-only"
	}
	if v == nil {
		return nil, ""
	}
	switch f.Type {
	case typeText, typeLongText:
		return asString(v)
	case typeEmail:
		return checkedString(v, func(s string) bool { return s == "" || emailPattern.MatchString(s) }, "must be a valid email address")
	case typeURL:
		return checkedString(v, func(s string) bool { return s == "" || urlPattern.MatchString(s) }, "must be a valid URL")
	case typePhone:
		return checkedString(v, func(s string) bool { return s == "" || phonePattern.MatchString(s) }, "must be a valid phone number")
	case typeNumber:
		return wholeNumber(v, math.MinInt32, math.MaxInt32)
	case typeRating:
		return wholeNumber(v, 0, int64(f.RatingMax))
	case typeYear:
		return wholeNumber(v, 1, 9999)
	case typeDecimal, typeCurrency, typePercent:
		n, ok := v.(json.Number)
		if !ok {
			return nil, "must be a number"
		}
		if _, err := n.Float64(); err != nil {
			return nil, "must be a number"
		}
		return n.String(), ""
	case typeDuration:
		value, issue := wholeNumber(v, 0, math.MaxInt32)
		if issue != "" {
			return nil, "must be a whole number of minutes, 0 or more"
		}
		// Stored as text, as the app stores it
		return strconv.FormatInt(value.(int64), 10), ""
	case typeCheckbox:
		b, ok := v.(bool)
		if !ok {
			return nil, "must be true or false"
		}
		return b, ""
	case typeDate:
		s, ok := v.(string)
		if !ok {
			return nil, "must be a date like 2026-11-03"
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return nil, "must be a date like 2026-11-03"
		}
		return s, ""
	case typeDateTime:
		s, ok := v.(string)
		if !ok {
			return nil, "must be a date and time like 2026-11-03T10:15:00Z"
		}
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return nil, "must be a date and time like 2026-11-03T10:15:00Z"
		}
		// Stored without a time zone, in UTC
		return t.UTC(), ""
	case typeTime:
		return checkedString(v, timePattern.MatchString, "must be a time like 10:15 (24-hour)")
	case typeSelect:
		s, ok := v.(string)
		if !ok {
			return nil, "must be the name of one of the field's options"
		}
		if !f.hasOption(s) {
			return nil, fmt.Sprintf("%q is not one of the field's options", s)
		}
		return s, ""
	case typeMultiSelect:
		list, ok := v.([]interface{})
		if !ok {
			return nil, "must be a list of option names"
		}
		names := make(pq.StringArray, 0, len(list))
		for _, item := range list {
			s, ok := item.(string)
			if !ok {
				return nil, "must be a list of option names"
			}
			if !f.hasOption(s) {
				return nil, fmt.Sprintf("%q is not one of the field's options", s)
			}
			names = append(names, s)
		}
		return names, ""
	case typeJSON:
		// Stored as text, as the app stores it
		b, err := json.Marshal(v)
		if err != nil {
			return nil, "must be valid JSON"
		}
		return string(b), ""
	case typeUUID:
		value, err := tableServices.UUIDCellValue(v)
		if err != nil {
			return nil, "must be a valid UUID"
		}
		return value, ""
	}
	return nil, "can't be written through the API"
}

func (f *field) hasOption(name string) bool {
	for _, o := range f.Options {
		if o == name {
			return true
		}
	}
	return false
}

func asString(v interface{}) (interface{}, string) {
	s, ok := v.(string)
	if !ok {
		return nil, "must be a string"
	}
	return s, ""
}

func checkedString(v interface{}, valid func(string) bool, issue string) (interface{}, string) {
	s, ok := v.(string)
	if !ok || !valid(s) {
		return nil, issue
	}
	return s, ""
}

func wholeNumber(v interface{}, min, max int64) (interface{}, string) {
	n, ok := v.(json.Number)
	if !ok {
		return nil, "must be a whole number"
	}
	i, err := n.Int64()
	if err != nil {
		return nil, "must be a whole number"
	}
	if i < min || i > max {
		return nil, fmt.Sprintf("must be between %d and %d", min, max)
	}
	return i, ""
}
