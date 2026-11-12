// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/aptlogica/go-postgres-rest/pkg/models"
)

// The where filter reads like a spreadsheet filter: "show records where all / any / none of these are true".
//
//	group:     {"match": "all" | "any" | "none", "conditions": [ ...groups or conditions... ]}
//	condition: {"field": "Status", "op": "eq", "value": "Open"}
//
// match defaults to all and op to eq. Each condition is checked like a filter[...] parameter, so the same
// operators work on the same field types.

const (
	maxWhereDepth      = 5  // groups inside groups
	maxWhereConditions = 50 // conditions in the whole tree
)

const whereExample = `{"match": "all", "conditions": [{"field": "Status", "value": "Open"}]}`

// parseWhereParam reads the where query parameter: the tree as JSON. ok is false when it filters nothing:
// empty, null, {} or a group without conditions.
func parseWhereParam(raw string, fields []*field, links linkResolver) (term filterTerm, ok bool, err error) {
	if strings.TrimSpace(raw) == "" {
		return filterTerm{}, false, nil
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil || dec.More() {
		return filterTerm{}, false, badRequest("where must be JSON, like " + whereExample + ".")
	}
	if emptyWhere(v) {
		return filterTerm{}, false, nil
	}
	term, err = parseWhere(v, fields, links)
	return term, err == nil, err
}

// emptyWhere tells whether a where tree filters nothing: null, {}, or a group without conditions
// ({"match": "all"}, {"conditions": []}). A group without conditions inside the tree is still an error.
func emptyWhere(v interface{}) bool {
	if v == nil {
		return true
	}
	obj, ok := v.(map[string]interface{})
	if !ok {
		return false
	}
	for k, val := range obj {
		switch k {
		case "match":
			if s, _ := val.(string); s != "all" && s != "any" && s != "none" {
				return false // reported by the parser
			}
		case "conditions":
			if list, isList := val.([]interface{}); val != nil && (!isList || len(list) > 0) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// parseWhere reads a where tree decoded with json.Number numbers. The tree may also be a single condition.
func parseWhere(v interface{}, fields []*field, links linkResolver) (filterTerm, error) {
	p := &whereParser{fields: fields, links: links}
	term, err := p.node(v, "where", 1)
	if err != nil {
		return filterTerm{}, err
	}
	term.key = "where:" + term.key
	return term, nil
}

type whereParser struct {
	fields []*field
	links  linkResolver
	count  int
}

func (p *whereParser) node(v interface{}, path string, depth int) (filterTerm, error) {
	obj, ok := v.(map[string]interface{})
	_, isCondition := obj["field"]
	_, hasConditions := obj["conditions"]
	_, hasMatch := obj["match"]
	isGroup := hasConditions || hasMatch
	switch {
	case !ok || !isCondition && !isGroup:
		return filterTerm{}, badRequest(fmt.Sprintf(
			`%s must be a group {"match": "all", "conditions": [...]} or a condition {"field": "Status", "op": "eq", "value": "Open"}.`, path))
	case isCondition && isGroup:
		return filterTerm{}, badRequest(path + ": a condition has field, op and value; a group has match and conditions. Use one or the other.")
	case isCondition:
		return p.condition(obj, path)
	}
	return p.group(obj, path, depth)
}

func (p *whereParser) group(obj map[string]interface{}, path string, depth int) (filterTerm, error) {
	if depth > maxWhereDepth {
		return filterTerm{}, badRequest(fmt.Sprintf("%s: groups can be nested at most %d deep.", path, maxWhereDepth))
	}
	if err := onlyKeys(obj, path, "match", "conditions"); err != nil {
		return filterTerm{}, err
	}
	match := "all"
	if raw, ok := obj["match"]; ok {
		s, _ := raw.(string)
		if s != "all" && s != "any" && s != "none" {
			return filterTerm{}, badRequest(path + ".match must be all, any or none.")
		}
		match = s
	}
	list, ok := obj["conditions"].([]interface{})
	if !ok || len(list) == 0 {
		return filterTerm{}, badRequest(path + ".conditions must be a list with at least one condition.")
	}

	node := models.ComplexFilter{Logic: "AND", Groups: make([]models.ComplexFilter, len(list))}
	keys := make([]string, len(list))
	for i, item := range list {
		term, err := p.node(item, fmt.Sprintf("%s.conditions[%d]", path, i), depth+1)
		if err != nil {
			return filterTerm{}, err
		}
		node.Groups[i] = term.node
		keys[i] = term.key
	}
	switch match {
	case "any":
		node.Logic = "OR"
	case "none":
		// none of them: NOT (a OR b), where an empty value counts as not matching
		node.Logic = "OR"
		node.Not = true
	}
	// The order of conditions doesn't change the result, so it doesn't change the cursor hash either
	sort.Strings(keys)
	return filterTerm{key: match + "(" + strings.Join(keys, ",") + ")", node: node}, nil
}

func (p *whereParser) condition(obj map[string]interface{}, path string) (filterTerm, error) {
	p.count++
	if p.count > maxWhereConditions {
		return filterTerm{}, badRequest(fmt.Sprintf("where: at most %d conditions.", maxWhereConditions))
	}
	if err := onlyKeys(obj, path, "field", "op", "value"); err != nil {
		return filterTerm{}, err
	}
	ref, _ := obj["field"].(string)
	if strings.TrimSpace(ref) == "" {
		return filterTerm{}, badRequest(path + ".field must be a field name or ID.")
	}
	op := "eq"
	if raw, ok := obj["op"]; ok {
		s, ok := raw.(string)
		if !ok || s == "" {
			return filterTerm{}, badRequest(path + ".op must be an operator, like eq or contains.")
		}
		op = s
	}

	c := filterCondition{label: path, ref: ref, op: op}
	raw, hasValue := obj["value"]
	switch v := raw.(type) {
	case nil:
		if hasValue {
			return filterTerm{}, badRequest(path + `.value can't be null. To find empty fields use "op": "is_null".`)
		}
		if op != "is_null" {
			return filterTerm{}, badRequest(path + ".value is missing.")
		}
		c.value = "true" // {"field": "Status", "op": "is_null"} finds empty fields
	case []interface{}:
		c.hasList = true
		c.list = make([]string, len(v))
		for i, item := range v {
			s, ok := scalarText(item)
			if !ok {
				return filterTerm{}, badRequest(fmt.Sprintf("%s.value[%d] must be text, a number or true/false.", path, i))
			}
			c.list[i] = s
		}
	default:
		s, ok := scalarText(v)
		if !ok {
			return filterTerm{}, badRequest(path + ".value must be text, a number, true/false, or a list for in.")
		}
		c.value = s
	}
	return buildCondition(p.fields, c, p.links)
}

// scalarText is a JSON string, number or boolean as the text a filter[...] parameter would hold
func scalarText(v interface{}) (string, bool) {
	switch v := v.(type) {
	case string:
		return v, true
	case json.Number:
		return v.String(), true
	case bool:
		return strconv.FormatBool(v), true
	}
	return "", false
}
