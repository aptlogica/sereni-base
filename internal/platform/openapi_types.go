// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

// schemaObject is an OpenAPI schema, parameter or other spec object
type schemaObject = map[string]any

const recordIDRef = "#/components/schemas/RecordID"

func schemaRef(target string) schemaObject { return schemaObject{"$ref": target} }

// fieldSchema is the OpenAPI schema of a field's value as the API returns it. Links are built by the caller,
// which needs the linked table; lookups have no schema of their own (they are inside link objects).
func fieldSchema(f *field) schemaObject {
	var s schemaObject
	switch f.Type {
	case typeText, typeLongText, typePhone, typeFormula, typeCreatedBy, typeLastModifiedBy:
		s = schemaObject{"type": "string"}
	case typeEmail:
		s = schemaObject{"type": "string", "format": "email"}
	case typeURL:
		// No format: uri, because the API accepts URLs without a scheme (example.com)
		s = schemaObject{"type": "string", "example": "https://example.com"}
	case typeNumber:
		s = schemaObject{"type": "integer", "format": "int32"}
	case typeDecimal, typeCurrency, typePercent:
		s = schemaObject{"type": "number", "description": "Sent and returned with all its digits; parsing into a float may round it."}
	case typeRating:
		s = schemaObject{"type": "integer", "minimum": 0, "maximum": f.RatingMax}
	case typeYear:
		s = schemaObject{"type": "integer", "minimum": 1, "maximum": 9999}
	case typeDuration:
		s = schemaObject{"type": "integer", "minimum": 0, "description": "Minutes"}
	case typeDate:
		s = schemaObject{"type": "string", "format": "date"}
	case typeDateTime, typeCreatedTime, typeLastModifiedTime:
		s = schemaObject{"type": "string", "format": "date-time"}
	case typeTime:
		s = schemaObject{"type": "string", "pattern": timePattern.String(), "example": "10:15"}
	case typeCheckbox:
		s = schemaObject{"type": "boolean"}
	case typeSelect:
		s = schemaObject{"type": "string"}
		if len(f.Options) > 0 {
			s["enum"] = f.Options
		}
	case typeMultiSelect:
		items := schemaObject{"type": "string"}
		if len(f.Options) > 0 {
			items["enum"] = f.Options
		}
		s = schemaObject{"type": "array", "items": items}
	case typeUUID:
		s = schemaObject{"type": "string", "format": "uuid"}
	case typeAttachment:
		s = schemaObject{"type": "array", "items": schemaObject{
			"type": "object",
			"properties": schemaObject{
				"id":   schemaObject{},
				"url":  schemaObject{"type": "string"},
				"name": schemaObject{"type": "string"},
				"size": schemaObject{"type": "integer"},
				"type": schemaObject{"type": "string"},
			},
		}}
	case typeUser:
		if f.Multiple {
			s = schemaObject{"type": "array", "items": schemaObject{"type": "string"}}
		} else {
			s = schemaObject{"type": "string"}
		}
	default:
		// json is any JSON value; unknown types are returned as stored
		s = schemaObject{}
	}
	if _, typed := s["type"]; typed && s["type"] != "array" {
		s["nullable"] = true
	}
	return s
}

// linkWriteSchema is a link field's value in a request: record IDs, or objects with an id
func linkWriteSchema(f *field) schemaObject {
	s := schemaObject{"type": "array", "items": schemaObject{"oneOf": []any{
		schemaRef(recordIDRef),
		schemaObject{"type": "object", "required": []string{"id"}, "properties": schemaObject{"id": schemaRef(recordIDRef)}},
	}}}
	if !f.LinkMulti {
		s["maxItems"] = 1
	}
	return s
}

var (
	scalarOperators = []string{"eq", "neq", "gt", "gte", "lt", "lte", "in", "is_null"}
	textOperators   = []string{"eq", "neq", "gt", "gte", "lt", "lte", "in", "contains", "is_null"}
)

// filterOperators lists the operators buildFilter accepts for a field; none when it can't be filtered
func filterOperators(f *field) []string {
	switch {
	case f.Type == typeMultiSelect || f.Type == typeLinks:
		return []string{"contains", "is_null"}
	case f.Type == typeJSON || f.Type == typeAttachment || f.Type == typeLookup:
		return nil
	case f.Type == typeCheckbox:
		return []string{"eq", "neq", "is_null"}
	case textTypes[f.Type]:
		return textOperators
	case numberTypes[f.Type], f.Type == typeDate, timeTypes[f.Type]:
		return scalarOperators
	}
	return nil
}
