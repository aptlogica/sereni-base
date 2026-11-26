package platform

import "testing"

func TestParseColumnInputTakesDefaultAndIDGenerationFromMeta(t *testing.T) {
	body := map[string]interface{}{
		"title": "Ref", "uidt": "uuid",
		"meta":        map[string]interface{}{"default": "x", "id_generation": map[string]interface{}{"mode": "auto", "version": "v4", "backfill": true}, "other": 1},
		"constraints": map[string]interface{}{"unique": true},
	}
	in, details, err := parseColumnInput(body, "body", true)
	if err != nil || len(details) > 0 {
		t.Fatalf("unexpected errors: %v %v", err, details)
	}
	if !in.Constraints.hasDefault || in.Constraints.Default != "x" {
		t.Errorf("default not read from meta: %+v", in.Constraints)
	}
	ig := in.Constraints.IDGeneration
	if ig == nil || ig.Mode != "auto" || ig.Version != "v4" || !ig.Backfill {
		t.Errorf("id_generation not read from meta: %+v", ig)
	}
	if in.Constraints.Unique == nil || !*in.Constraints.Unique {
		t.Errorf("unique lost")
	}
	if _, has := in.Meta["default"]; has {
		t.Errorf("default must not stay in meta")
	}
	if _, has := in.Meta["id_generation"]; has {
		t.Errorf("id_generation must not stay in meta")
	}
	if in.Meta["other"] != 1 {
		t.Errorf("other meta keys must be kept: %v", in.Meta)
	}
}

func TestParseColumnInputRefusesSettingsInConstraints(t *testing.T) {
	for _, key := range []string{"default", "id_generation"} {
		body := map[string]interface{}{
			"title": "A", "uidt": "text",
			"constraints": map[string]interface{}{key: "x"},
		}
		_, details, err := parseColumnInput(body, "body", true)
		if err != nil {
			t.Fatal(err)
		}
		if len(details) != 1 || details[0].Field != "body.constraints."+key {
			t.Errorf("%s in constraints should be refused, got %+v", key, details)
		}
	}
}

func TestParseColumnInputBadIDGenerationInMeta(t *testing.T) {
	body := map[string]interface{}{
		"title": "A", "uidt": "uuid",
		"meta": map[string]interface{}{"id_generation": "auto"},
	}
	_, details, _ := parseColumnInput(body, "body", true)
	if len(details) != 1 || details[0].Field != "body.meta.id_generation" {
		t.Errorf("got %+v", details)
	}
}
