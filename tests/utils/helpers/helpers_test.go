package tests

import (
	"serenibase/internal/utils/helpers"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestGenerateID tests the GenerateID function
func TestGenerateID(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 8", 8},
		{"length 16", 16},
		{"length 32", 32},
		{"length 0", 0},
		{"length 1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := helpers.GenerateID(tt.length)
			expectedLen := tt.length * 2
			if len(id) != expectedLen {
				t.Errorf("GenerateID(%d) length = %d, want %d", tt.length, len(id), expectedLen)
			}
			for _, char := range id {
				if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
					t.Errorf("GenerateID(%d) contains invalid hex character: %c", tt.length, char)
				}
			}
		})
	}

	t.Run("uniqueness", func(t *testing.T) {
		id1 := helpers.GenerateID(16)
		id2 := helpers.GenerateID(16)
		if id1 == id2 {
			t.Errorf("GenerateID should generate unique IDs, got %s and %s", id1, id2)
		}
	})
}

// TestConvertToString tests the ConvertToString function
func TestConvertToString(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"nil value", nil, ""},
		{"string", "hello", "hello"},
		{"int", 123, "123"},
		{"float64", float64(12.34), "12.34"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"time", now, now.Format(time.RFC3339)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.ConvertToString(tt.value)
			if result != tt.expected {
				t.Errorf("ConvertToString(%v) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

// TestIsEmpty tests the IsEmpty function
func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"empty slice", []int{}, true},
		{"non-empty slice", []int{1, 2, 3}, false},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"nil pointer", (*int)(nil), true},
		{"non-nil pointer", new(int), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.IsEmpty(tt.value)
			if result != tt.expected {
				t.Errorf("IsEmpty(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestContains tests the Contains function
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    interface{}
		element  interface{}
		expected bool
	}{
		{"int slice contains", []int{1, 2, 3, 4}, 3, true},
		{"int slice not contains", []int{1, 2, 3, 4}, 5, false},
		{"string slice contains", []string{"a", "b", "c"}, "b", true},
		{"string slice not contains", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []int{}, 1, false},
		{"not a slice", "not a slice", 1, false},
		{"nil slice", nil, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.Contains(tt.slice, tt.element)
			if result != tt.expected {
				t.Errorf("Contains(%v, %v) = %v, want %v", tt.slice, tt.element, result, tt.expected)
			}
		})
	}
}

// TestTruncateString tests the TruncateString function
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{"within length", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate needed", "hello world", 8, "hello..."},
		{"very short", "hello", 2, "he"},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.TruncateString(tt.input, tt.length)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.length, result, tt.expected)
			}
		})
	}
}

// TestToSnakeCase tests the ToSnakeCase function
func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"PascalCase", "PascalCase", "pascal_case"},
		{"camelCase", "camelCase", "camel_case"},
		{"With Spaces", "With Spaces", "with_spaces"},
		{"With-Dashes", "With-Dashes", "with_dashes"},
		{"lowercase", "lowercase", "lowercase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestStringPtr tests the StringPtr function
func TestStringPtr(t *testing.T) {
	s := "test"
	ptr := helpers.StringPtr(s)

	if ptr == nil {
		t.Fatal("StringPtr returned nil")
	}

	if *ptr != s {
		t.Errorf("StringPtr value = %q, want %q", *ptr, s)
	}
}

// TestFloat64Ptr tests the Float64Ptr function
func TestFloat64Ptr(t *testing.T) {
	f := 3.14
	ptr := helpers.Float64Ptr(f)

	if ptr == nil {
		t.Fatal("Float64Ptr returned nil")
	}

	if *ptr != f {
		t.Errorf("Float64Ptr value = %f, want %f", *ptr, f)
	}
}

// TestBoolPtr tests the BoolPtr function
func TestBoolPtr(t *testing.T) {
	b := true
	ptr := helpers.BoolPtr(b)

	if ptr == nil {
		t.Fatal("BoolPtr returned nil")
	}

	if *ptr != b {
		t.Errorf("BoolPtr value = %v, want %v", *ptr, b)
	}
}

// TestTimeAgo tests the TimeAgo function
func TestTimeAgo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		time     time.Time
		contains string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"minutes ago", now.Add(-5 * time.Minute), "minute"},
		{"hours ago", now.Add(-2 * time.Hour), "hour"},
		{"days ago", now.Add(-2 * 24 * time.Hour), "day"},
		{"weeks ago", now.Add(-10 * 24 * time.Hour), "week"},
		{"months ago", now.Add(-45 * 24 * time.Hour), "month"},
		{"years ago", now.Add(-400 * 24 * time.Hour), "year"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.TimeAgo(tt.time)
			if result == "" {
				t.Error("TimeAgo returned empty string")
			}
		})
	}
}

// TestRemoveDuplicates tests the RemoveDuplicates function
func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{"int slice with duplicates", []int{1, 2, 2, 3, 3, 3, 4}, []int{1, 2, 3, 4}},
		{"string slice with duplicates", []string{"a", "b", "b", "c"}, []string{"a", "b", "c"}},
		{"no duplicates", []int{1, 2, 3}, []int{1, 2, 3}},
		{"empty slice", []int{}, []int{}},
		{"all same", []int{5, 5, 5, 5}, []int{5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.RemoveDuplicates(tt.input)
			// Simple length check for validation
			switch v := result.(type) {
			case []int:
				expected := tt.expected.([]int)
				if len(v) != len(expected) {
					t.Errorf("RemoveDuplicates length = %d, want %d", len(v), len(expected))
				}
			case []string:
				expected := tt.expected.([]string)
				if len(v) != len(expected) {
					t.Errorf("RemoveDuplicates length = %d, want %d", len(v), len(expected))
				}
			}
		})
	}

	t.Run("not a slice", func(t *testing.T) {
		result := helpers.RemoveDuplicates("not a slice")
		if result != "not a slice" {
			t.Error("RemoveDuplicates should return input as-is for non-slice")
		}
	})
}

// TestFormatFileSize tests the FormatFileSize function
func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"bytes", 500, "500 B"},
		{"kilobytes", 1024, "1.0 KB"},
		{"megabytes", 1024 * 1024, "1.0 MB"},
		{"gigabytes", 1024 * 1024 * 1024, "1.0 GB"},
		{"terabytes", 1024 * 1024 * 1024 * 1024, "1.0 TB"},
		{"zero", 0, "0 B"},
		{"mixed", 1536, "1.5 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.FormatFileSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatFileSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

// TestStringToSlice tests the StringToSlice function
func TestStringToSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple", "a,b,c", []string{"a", "b", "c"}},
		{"with spaces", "a, b, c", []string{"a", "b", "c"}},
		{"empty string", "", []string{}},
		{"single item", "only", []string{"only"}},
		{"with empty items", "a,,b", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.StringToSlice(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("StringToSlice(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("StringToSlice(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

// TestMapKeys tests the MapKeys function
func TestMapKeys(t *testing.T) {
	t.Run("string keys", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2, "c": 3}
		keys := helpers.MapKeys(m)
		if len(keys) != 3 {
			t.Errorf("MapKeys length = %d, want 3", len(keys))
		}
	})

	t.Run("int keys", func(t *testing.T) {
		m := map[int]string{1: "a", 2: "b"}
		keys := helpers.MapKeys(m)
		if len(keys) != 2 {
			t.Errorf("MapKeys length = %d, want 2", len(keys))
		}
	})

	t.Run("empty map", func(t *testing.T) {
		m := map[string]int{}
		keys := helpers.MapKeys(m)
		if len(keys) != 0 {
			t.Errorf("MapKeys length = %d, want 0", len(keys))
		}
	})

	t.Run("not a map", func(t *testing.T) {
		keys := helpers.MapKeys("not a map")
		if keys != nil {
			t.Error("MapKeys should return nil for non-map")
		}
	})
}

// TestMapValues tests the MapValues function
func TestMapValues(t *testing.T) {
	t.Run("string values", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2, "c": 3}
		values := helpers.MapValues(m)
		if len(values) != 3 {
			t.Errorf("MapValues length = %d, want 3", len(values))
		}
	})

	t.Run("empty map", func(t *testing.T) {
		m := map[string]int{}
		values := helpers.MapValues(m)
		if len(values) != 0 {
			t.Errorf("MapValues length = %d, want 0", len(values))
		}
	})

	t.Run("not a map", func(t *testing.T) {
		values := helpers.MapValues(123)
		if values != nil {
			t.Error("MapValues should return nil for non-map")
		}
	})
}

// TestSliceToString tests the SliceToString function
func TestSliceToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"int slice", []int{1, 2, 3}, "1, 2, 3"},
		{"string slice", []string{"a", "b", "c"}, "a, b, c"},
		{"empty slice", []int{}, ""},
		{"single item", []string{"only"}, "only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.SliceToString(tt.input)
			if result != tt.expected {
				t.Errorf("SliceToString(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}

	t.Run("not a slice", func(t *testing.T) {
		result := helpers.SliceToString("not a slice")
		if result != "" {
			t.Error("SliceToString should return empty string for non-slice")
		}
	})
}

// TestReverse tests the Reverse function
func TestReverse(t *testing.T) {
	t.Run("int slice", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 5}
		helpers.Reverse(slice)
		expected := []int{5, 4, 3, 2, 1}
		for i, v := range slice {
			if v != expected[i] {
				t.Errorf("Reverse()[%d] = %d, want %d", i, v, expected[i])
			}
		}
	})

	t.Run("string slice", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		helpers.Reverse(slice)
		expected := []string{"c", "b", "a"}
		for i, v := range slice {
			if v != expected[i] {
				t.Errorf("Reverse()[%d] = %q, want %q", i, v, expected[i])
			}
		}
	})

	t.Run("single element", func(t *testing.T) {
		slice := []int{42}
		helpers.Reverse(slice)
		if slice[0] != 42 {
			t.Error("Reverse should not change single element slice")
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		slice := []int{}
		helpers.Reverse(slice)
		if len(slice) != 0 {
			t.Error("Reverse should not change empty slice")
		}
	})

	t.Run("not a slice", func(t *testing.T) {
		// Should not panic
		helpers.Reverse("not a slice")
	})
}

// TestMapToStruct tests the MapToStruct function
func TestMapToStruct(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid mapping",
			input: map[string]interface{}{
				"name":  "John Doe",
				"age":   30,
				"email": "john@example.com",
			},
			wantErr: false,
		},
		{
			name: "partial mapping",
			input: map[string]interface{}{
				"name": "Jane",
			},
			wantErr: false,
		},
		{
			name:    "empty map",
			input:   map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target TestStruct
			err := helpers.MapToStruct(tt.input, &target)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapToStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStructToStruct tests the StructToStruct function
func TestStructToStruct(t *testing.T) {
	type Source struct {
		Name  string
		Age   int
		Email string
	}

	type Target struct {
		Name  string
		Age   int
		Email string
	}

	tests := []struct {
		name    string
		src     Source
		wantErr bool
	}{
		{
			name: "full copy",
			src: Source{
				Name:  "John",
				Age:   30,
				Email: "john@example.com",
			},
			wantErr: false,
		},
		{
			name:    "empty struct",
			src:     Source{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target Target
			err := helpers.StructToStruct(&tt.src, &target)
			if (err != nil) != tt.wantErr {
				t.Errorf("StructToStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if target.Name != tt.src.Name {
					t.Errorf("StructToStruct() Name = %q, want %q", target.Name, tt.src.Name)
				}
				if target.Age != tt.src.Age {
					t.Errorf("StructToStruct() Age = %d, want %d", target.Age, tt.src.Age)
				}
			}
		})
	}
}

// TestPtr tests the generic Ptr function
func TestPtr(t *testing.T) {
	t.Run("int pointer", func(t *testing.T) {
		val := 42
		ptr := helpers.Ptr(val)
		if ptr == nil {
			t.Fatal("Ptr returned nil")
		}
		if *ptr != val {
			t.Errorf("Ptr value = %d, want %d", *ptr, val)
		}
	})

	t.Run("string pointer", func(t *testing.T) {
		val := "test"
		ptr := helpers.Ptr(val)
		if ptr == nil {
			t.Fatal("Ptr returned nil")
		}
		if *ptr != val {
			t.Errorf("Ptr value = %q, want %q", *ptr, val)
		}
	})

	t.Run("bool pointer", func(t *testing.T) {
		val := true
		ptr := helpers.Ptr(val)
		if ptr == nil {
			t.Fatal("Ptr returned nil")
		}
		if *ptr != val {
			t.Errorf("Ptr value = %v, want %v", *ptr, val)
		}
	})
}

// TestInterfaceToJSONString tests the InterfaceToJSONString function
func TestInterfaceToJSONString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		contains string
	}{
		{
			name:     "simple object",
			input:    map[string]interface{}{"key": "value"},
			contains: "key",
		},
		{
			name:     "nested object",
			input:    map[string]interface{}{"outer": map[string]interface{}{"inner": "value"}},
			contains: "outer",
		},
		{
			name:     "nil map",
			input:    nil,
			contains: "null",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			contains: "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.InterfaceToJSONString(tt.input)
			if result == "" {
				t.Error("InterfaceToJSONString returned empty string")
			}
		})
	}
}

// TestSplitAndTrim tests the SplitAndTrim function
func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		delimiter string
		expected  []string
	}{
		{"simple", "a,b,c", ",", []string{"a", "b", "c"}},
		{"with spaces", " a , b , c ", ",", []string{"a", "b", "c"}},
		{"empty string", "", ",", []string{}},
		{"single item", "only", ",", []string{"only"}},
		{"with empty items", "a,,b", ",", []string{"a", "b"}},
		{"different delimiter", "a|b|c", "|", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.SplitAndTrim(tt.input, tt.delimiter)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitAndTrim length = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("SplitAndTrim[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

// TestConvertToString_AllTypes tests all type conversions
func TestConvertToString_AllTypes(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{"int8", int8(127)},
		{"int16", int16(32767)},
		{"int32", int32(2147483647)},
		{"int64", int64(9223372036854775807)},
		{"uint", uint(42)},
		{"uint8", uint8(255)},
		{"uint16", uint16(65535)},
		{"uint32", uint32(4294967295)},
		{"uint64", uint64(18446744073709551615)},
		{"float32", float32(3.14)},
		{"complex type", struct{ X int }{X: 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.ConvertToString(tt.value)
			if result == "" {
				t.Errorf("ConvertToString(%v) returned empty string", tt.value)
			}
		})
	}
}

// TestIsEmpty_AllTypes tests IsEmpty with all type variations
func TestIsEmpty_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"int8 zero", int8(0), true},
		{"int8 non-zero", int8(1), false},
		{"int16 zero", int16(0), true},
		{"int32 zero", int32(0), true},
		{"int64 zero", int64(0), true},
		{"uint zero", uint(0), true},
		{"uint non-zero", uint(1), false},
		{"uint8 zero", uint8(0), true},
		{"uint16 zero", uint16(0), true},
		{"uint32 zero", uint32(0), true},
		{"uint64 zero", uint64(0), true},
		{"uintptr zero", uintptr(0), true},
		{"float32 zero", float32(0), true},
		{"float32 non-zero", float32(1.5), false},
		{"interface nil", (interface{})(nil), true},
		{"channel nil", (chan int)(nil), true},
		{"channel empty", make(chan int), true},
		{"struct", struct{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.IsEmpty(tt.value)
			if result != tt.expected {
				t.Errorf("IsEmpty(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestMapToStruct_ErrorCases tests MapToStruct error handling
func TestMapToStruct_ErrorCases(t *testing.T) {
	type InvalidStruct struct {
		Channel chan int `json:"channel"`
	}

	t.Run("invalid json marshal", func(t *testing.T) {
		input := map[string]interface{}{
			"channel": make(chan int),
		}
		var target InvalidStruct
		err := helpers.MapToStruct(input, &target)
		if err == nil {
			t.Error("MapToStruct should error on invalid marshal")
		}
	})
}

// TestStructToStruct_ComplexTypes tests StructToStruct with complex types
func TestStructToStruct_ComplexTypes(t *testing.T) {
	type SourceWithTime struct {
		Name      string
		CreatedAt time.Time
		UpdatedAt *time.Time
	}

	type TargetWithTime struct {
		Name      string
		CreatedAt time.Time
		UpdatedAt *time.Time
	}

	now := time.Now()
	tests := []struct {
		name    string
		src     SourceWithTime
		wantErr bool
	}{
		{
			name: "with time pointers",
			src: SourceWithTime{
				Name:      "Test",
				CreatedAt: now,
				UpdatedAt: &now,
			},
			wantErr: false,
		},
		{
			name: "with nil time pointer",
			src: SourceWithTime{
				Name:      "Test",
				CreatedAt: now,
				UpdatedAt: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target TargetWithTime
			err := helpers.StructToStruct(&tt.src, &target)
			if (err != nil) != tt.wantErr {
				t.Errorf("StructToStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestInterfaceToJSONString_EdgeCases tests edge cases
func TestInterfaceToJSONString_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{
			name: "with array",
			input: map[string]interface{}{
				"items": []int{1, 2, 3},
			},
		},
		{
			name: "with boolean",
			input: map[string]interface{}{
				"active": true,
			},
		},
		{
			name: "with number",
			input: map[string]interface{}{
				"count": 42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.InterfaceToJSONString(tt.input)
			if result == "" || result == "null" {
				t.Errorf("InterfaceToJSONString returned '%s'", result)
			}
		})
	}
}

// TestInterfaceToJSONString_UnmarshalableType tests error case with unmarshalable type
func TestInterfaceToJSONString_UnmarshalableType(t *testing.T) {
	// Create a map with a channel which cannot be marshaled to JSON
	input := map[string]interface{}{
		"channel": make(chan int),
	}

	result := helpers.InterfaceToJSONString(input)
	if result != "null" {
		t.Errorf("InterfaceToJSONString with unmarshalable type should return 'null', got %s", result)
	}
}

// TestMapToStruct_UnmarshalError tests MapToStruct with unmarshal error
func TestMapToStruct_UnmarshalError(t *testing.T) {
	type Target struct {
		Value int `json:"value"`
	}

	// Create invalid data that will fail to unmarshal
	input := map[string]interface{}{
		"value": "not_a_number", // This should fail when trying to unmarshal into int
	}

	var target Target
	err := helpers.MapToStruct(input, &target)
	if err == nil {
		t.Error("MapToStruct should return error for type mismatch")
	}
}

// TestStructToStruct_WithConverters tests StructToStruct with time and UUID conversions
func TestStructToStruct_WithConverters(t *testing.T) {
	now := time.Now()
	testUUID := "550e8400-e29b-41d4-a716-446655440000"

	type SourceWithConverters struct {
		Name      string
		CreatedAt time.Time
		UpdatedAt *time.Time
		ID        string
	}

	type TargetWithConverters struct {
		Name      string
		CreatedAt time.Time
		UpdatedAt *time.Time
		ID        string
	}

	tests := []struct {
		name    string
		src     SourceWithConverters
		wantErr bool
	}{
		{
			name: "with time conversions",
			src: SourceWithConverters{
				Name:      "Test",
				CreatedAt: now,
				UpdatedAt: &now,
				ID:        testUUID,
			},
			wantErr: false,
		},
		{
			name: "with nil time pointer",
			src: SourceWithConverters{
				Name:      "Test",
				CreatedAt: now,
				UpdatedAt: nil,
				ID:        testUUID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target TargetWithConverters
			err := helpers.StructToStruct(&tt.src, &target)
			if (err != nil) != tt.wantErr {
				t.Errorf("StructToStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if target.Name != tt.src.Name {
					t.Errorf("Name = %v, want %v", target.Name, tt.src.Name)
				}
				if !target.CreatedAt.Equal(tt.src.CreatedAt) {
					t.Errorf("CreatedAt = %v, want %v", target.CreatedAt, tt.src.CreatedAt)
				}
			}
		})
	}
}

// TestStructToStruct_TimeToStringPtr tests time.Time to *string conversion
func TestStructToStruct_TimeToStringPtr(t *testing.T) {
	now := time.Now()

	type SourceWithTime struct {
		Date *time.Time
	}

	type TargetWithString struct {
		Date *string
	}

	src := SourceWithTime{
		Date: &now,
	}

	var target TargetWithString
	err := helpers.StructToStruct(&src, &target)
	if err != nil {
		t.Fatalf("StructToStruct() error = %v", err)
	}

	if target.Date == nil {
		t.Error("Date should not be nil")
	} else {
		// Should be in yyyy-mm-dd format
		expected := now.Format("2006-01-02")
		if *target.Date != expected {
			t.Errorf("Date = %v, want %v", *target.Date, expected)
		}
	}
}

// TestStructToStruct_UUIDConversions tests UUID string conversions
func TestStructToStruct_UUIDConversions(t *testing.T) {
	testUUIDStr := "550e8400-e29b-41d4-a716-446655440000"
	testUUID, _ := uuid.Parse(testUUIDStr)

	type SourceWithUUID struct {
		ID uuid.UUID
	}

	type TargetWithString struct {
		ID string
	}

	src := SourceWithUUID{
		ID: testUUID,
	}

	var target TargetWithString
	err := helpers.StructToStruct(&src, &target)
	if err != nil {
		t.Fatalf("StructToStruct() error = %v", err)
	}

	if target.ID != testUUIDStr {
		t.Errorf("ID = %v, want %v", target.ID, testUUIDStr)
	}

	// Test reverse conversion
	type TargetWithUUID struct {
		ID uuid.UUID
	}

	var target2 TargetWithUUID
	err = helpers.StructToStruct(&target, &target2)
	if err != nil {
		t.Fatalf("StructToStruct() reverse error = %v", err)
	}

	if target2.ID != testUUID {
		t.Errorf("ID = %v, want %v", target2.ID, testUUID)
	}
}

// TestStructToStruct_InvalidUUIDConversion tests invalid UUID string conversion
func TestStructToStruct_InvalidUUIDConversion(t *testing.T) {
	type SourceWithInvalidUUID struct {
		ID string
	}

	type TargetWithUUID struct {
		ID uuid.UUID
	}

	src := SourceWithInvalidUUID{
		ID: "invalid-uuid-string",
	}

	var target TargetWithUUID
	err := helpers.StructToStruct(&src, &target)
	// Should return error for invalid UUID
	if err == nil {
		t.Error("StructToStruct() should return error for invalid UUID string")
	}
}

// TestStructToStruct_TimePtrToTimePtrConversions tests *time.Time to *time.Time
func TestStructToStruct_TimePtrToTimePtrConversions(t *testing.T) {
	now := time.Now()

	type SourceWithTimePtr struct {
		Date *time.Time
	}

	type TargetWithTimePtr struct {
		Date *time.Time
	}

	tests := []struct {
		name string
		src  SourceWithTimePtr
	}{
		{
			name: "non-nil time pointer",
			src: SourceWithTimePtr{
				Date: &now,
			},
		},
		{
			name: "nil time pointer",
			src: SourceWithTimePtr{
				Date: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target TargetWithTimePtr
			err := helpers.StructToStruct(&tt.src, &target)
			if err != nil {
				t.Fatalf("StructToStruct() error = %v", err)
			}

			if tt.src.Date == nil {
				if target.Date != nil {
					t.Error("target.Date should be nil when source is nil")
				}
			} else {
				if target.Date == nil {
					t.Error("target.Date should not be nil")
				} else if !target.Date.Equal(*tt.src.Date) {
					t.Errorf("Date = %v, want %v", *target.Date, *tt.src.Date)
				}
			}
		})
	}
}

// TestStructToStruct_TimeToTimePtr tests time.Time to *time.Time conversion
func TestStructToStruct_TimeToTimePtr(t *testing.T) {
	now := time.Now()

	type SourceWithTime struct {
		Date time.Time
	}

	type TargetWithTimePtr struct {
		Date *time.Time
	}

	src := SourceWithTime{
		Date: now,
	}

	var target TargetWithTimePtr
	err := helpers.StructToStruct(&src, &target)
	if err != nil {
		t.Fatalf("StructToStruct() error = %v", err)
	}

	if target.Date == nil {
		t.Error("Date should not be nil")
	} else if !target.Date.Equal(now) {
		t.Errorf("Date = %v, want %v", *target.Date, now)
	}
}

// TestStructToStruct_TimePtrToTime tests *time.Time to time.Time conversion
func TestStructToStruct_TimePtrToTime(t *testing.T) {
	now := time.Now()

	type SourceWithTimePtr struct {
		Date *time.Time
	}

	type TargetWithTime struct {
		Date time.Time
	}

	tests := []struct {
		name string
		src  SourceWithTimePtr
	}{
		{
			name: "non-nil time pointer",
			src: SourceWithTimePtr{
				Date: &now,
			},
		},
		{
			name: "nil time pointer",
			src: SourceWithTimePtr{
				Date: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target TargetWithTime
			err := helpers.StructToStruct(&tt.src, &target)
			if err != nil {
				t.Fatalf("StructToStruct() error = %v", err)
			}

			if tt.src.Date != nil {
				if !target.Date.Equal(*tt.src.Date) {
					t.Errorf("Date = %v, want %v", target.Date, *tt.src.Date)
				}
			} else {
				// When source is nil, target should be zero time
				if !target.Date.IsZero() {
					t.Error("Date should be zero when source is nil")
				}
			}
		})
	}
}

// TestStructToStruct_TimeToTime tests time.Time to time.Time pass-through
func TestStructToStruct_TimeToTime(t *testing.T) {
	now := time.Now()

	type SourceWithTime struct {
		Date time.Time
	}

	type TargetWithTime struct {
		Date time.Time
	}

	src := SourceWithTime{
		Date: now,
	}

	var target TargetWithTime
	err := helpers.StructToStruct(&src, &target)
	if err != nil {
		t.Fatalf("StructToStruct() error = %v", err)
	}

	if !target.Date.Equal(now) {
		t.Errorf("Date = %v, want %v", target.Date, now)
	}
}
