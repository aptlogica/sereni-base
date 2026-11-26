package platform

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
)

func TestConstraintOutIsRequestShaped(t *testing.T) {
	where := map[string]interface{}{"all": []interface{}{map[string]interface{}{"column": "Age", "op": "gte", "value": 18.0}}}
	resp := dto.TableConstraintResponse{
		ID: "7d0a3c1e-0000-4000-8000-000000000001", Title: "Adults", ConstraintName: "ck_x", Status: dto.ConstraintStatusOK,
		Columns:    []dto.ConstraintColumn{{ID: "c1", Title: "Age"}},
		CheckInput: dto.CheckInput{Category: "compare", Condition: "(age >= 18)", Where: where},
	}
	out := constraintOutOf(resp, "check")
	raw, _ := json.Marshal(out)
	var got map[string]interface{}
	_ = json.Unmarshal(raw, &got)

	if got["title"] != "Adults" || got["type"] != "check" {
		t.Fatalf("title/type: %v", got)
	}
	if !reflect.DeepEqual(got["columns"], []interface{}{"Age"}) {
		t.Errorf("columns should be names: %v", got["columns"])
	}
	chk := got["check"].(map[string]interface{})
	if chk["category"] != "compare" || !reflect.DeepEqual(chk["where"], where) {
		t.Errorf("check should carry category and where: %v", chk)
	}

	// Sending the response back, as a PUT would, leaves only fields the request takes
	dropEchoedFields(got, "check", "compare")
	if err := onlyKeys(got, "body", "title", "name", "type", "columns", "check"); err != nil {
		t.Errorf("echoed response still has unknown keys: %v", err)
	}
	if _, has := got["type"]; has {
		t.Errorf("an unchanged type should be dropped")
	}
}

func TestDropEchoedFieldsAppMadeCheck(t *testing.T) {
	// A check made in the app has no where: repeating its category and SQL must not be an error
	body := map[string]interface{}{"title": "T", "check": map[string]interface{}{"category": "custom", "condition": "(a > 1)"}}
	dropEchoedFields(body, "check", "custom")
	if _, has := body["check"]; has {
		t.Errorf("check without where should be dropped: %v", body)
	}
	// A different category stays, so the missing where is reported
	body = map[string]interface{}{"check": map[string]interface{}{"category": "text"}}
	dropEchoedFields(body, "check", "custom")
	if _, has := body["check"]; !has {
		t.Errorf("a changed category must be kept")
	}
}
