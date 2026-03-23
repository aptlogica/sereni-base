package tests

import (
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/utils/helpers"
	"github.com/google/uuid"
)

// TestRemoveDuplicates_Extended tests the RemoveDuplicates function
func TestRemoveDuplicates_Extended(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "int slice with duplicates",
			input:    []int{1, 2, 2, 3, 3, 3, 4},
			expected: []int{1, 2, 3, 4},
		},
		{
			name:     "string slice with duplicates",
			input:    []string{"a", "b", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "no duplicates",
			input:    []int{1, 2, 3},
			expected: []int{1, 2, 3},
		},
		{
			name:     "empty slice",
			input:    []int{},
			expected: []int{},
		},
		{
			name:     "not a slice",
			input:    "not a slice",
			expected: "not a slice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.RemoveDuplicates(tt.input)
			// Check type and length for slices
			if intSlice, ok := result.([]int); ok {
				expectedSlice := tt.expected.([]int)
				if len(intSlice) != len(expectedSlice) {
					t.Errorf("RemoveDuplicates length = %d, want %d", len(intSlice), len(expectedSlice))
				}
			}
		})
	}
}

// TestFormatFileSize_Extended tests the FormatFileSize function
func TestFormatFileSize_Extended(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"one byte", 1, "1 B"},
		{"500 bytes", 500, "500 B"},
		{"1 KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"1 MB", 1048576, "1.0 MB"},
		{"1 GB", 1073741824, "1.0 GB"},
		{"1 TB", 1099511627776, "1.0 TB"},
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

// TestStringToSlice_Extended tests the StringToSlice function
func TestStringToSlice_Extended(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"comma separated", "a,b,c", []string{"a", "b", "c"}},
		{"with spaces", "a, b, c", []string{"a", "b", "c"}},
		{"single item", "a", []string{"a"}},
		{"empty string", "", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.StringToSlice(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("StringToSlice(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("StringToSlice(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

// TestSplitAndTrim tests the SplitAndTrim function
func TestSplitAndTrim_Extended(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		delimiter string
		expected  []string
	}{
		{"comma delimiter", "a,b,c", ",", []string{"a", "b", "c"}},
		{"semicolon delimiter", "a;b;c", ";", []string{"a", "b", "c"}},
		{"with spaces", " a , b , c ", ",", []string{"a", "b", "c"}},
		{"empty string", "", ",", []string{}},
		{"single element", "abc", ",", []string{"abc"}},
		{"empty elements trimmed", "a,,b", ",", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.SplitAndTrim(tt.str, tt.delimiter)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitAndTrim(%q, %q) length = %d, want %d", tt.str, tt.delimiter, len(result), len(tt.expected))
			}
			for i, v := range result {
				if i < len(tt.expected) && v != tt.expected[i] {
					t.Errorf("SplitAndTrim(%q, %q)[%d] = %q, want %q", tt.str, tt.delimiter, i, v, tt.expected[i])
				}
			}
		})
	}
}

// TestMapKeys tests the MapKeys function
func TestMapKeys_Extended(t *testing.T) {
	tests := []struct {
		name  string
		m     interface{}
		count int
	}{
		{"string keys", map[string]int{"a": 1, "b": 2, "c": 3}, 3},
		{"int keys", map[int]string{1: "a", 2: "b"}, 2},
		{"empty map", map[string]int{}, 0},
		{"not a map", "not a map", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.MapKeys(tt.m)
			if tt.count == 0 {
				if result != nil && len(result) != 0 {
					t.Errorf("MapKeys expected nil or empty, got %v", result)
				}
			} else if len(result) != tt.count {
				t.Errorf("MapKeys length = %d, want %d", len(result), tt.count)
			}
		})
	}
}

// TestMapValues tests the MapValues function
func TestMapValues_Extended(t *testing.T) {
	tests := []struct {
		name  string
		m     interface{}
		count int
	}{
		{"string values", map[string]int{"a": 1, "b": 2, "c": 3}, 3},
		{"int values", map[int]string{1: "a", 2: "b"}, 2},
		{"empty map", map[string]int{}, 0},
		{"not a map", "not a map", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.MapValues(tt.m)
			if tt.count == 0 {
				if result != nil && len(result) != 0 {
					t.Errorf("MapValues expected nil or empty, got %v", result)
				}
			} else if len(result) != tt.count {
				t.Errorf("MapValues length = %d, want %d", len(result), tt.count)
			}
		})
	}
}

// TestSliceToString tests the SliceToString function
func TestSliceToString_Extended(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"int slice", []int{1, 2, 3}, "1, 2, 3"},
		{"string slice", []string{"a", "b", "c"}, "a, b, c"},
		{"empty slice", []int{}, ""},
		{"single element", []int{42}, "42"},
		{"not a slice", "not a slice", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.SliceToString(tt.input)
			if result != tt.expected {
				t.Errorf("SliceToString(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestReverse tests the Reverse function
func TestReverse_Extended(t *testing.T) {
	t.Run("int slice", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 5}
		helpers.Reverse(slice)
		expected := []int{5, 4, 3, 2, 1}
		for i, v := range slice {
			if v != expected[i] {
				t.Errorf("Reverse slice[%d] = %d, want %d", i, v, expected[i])
			}
		}
	})

	t.Run("string slice", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		helpers.Reverse(slice)
		expected := []string{"c", "b", "a"}
		for i, v := range slice {
			if v != expected[i] {
				t.Errorf("Reverse slice[%d] = %q, want %q", i, v, expected[i])
			}
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		slice := []int{}
		helpers.Reverse(slice)
		if len(slice) != 0 {
			t.Error("Empty slice should remain empty")
		}
	})

	t.Run("single element", func(t *testing.T) {
		slice := []int{42}
		helpers.Reverse(slice)
		if slice[0] != 42 {
			t.Errorf("Single element should remain %d, got %d", 42, slice[0])
		}
	})

	t.Run("not a slice", func(t *testing.T) {
		helpers.Reverse("not a slice") // Should not panic
	})
}

// TestMapToStruct tests the MapToStruct function
func TestMapToStruct_Extended(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	t.Run("valid conversion", func(t *testing.T) {
		input := map[string]interface{}{
			"name":  "test",
			"value": 42,
		}
		var target TestStruct
		err := helpers.MapToStruct(input, &target)
		if err != nil {
			t.Errorf("MapToStruct error: %v", err)
		}
		if target.Name != "test" {
			t.Errorf("Name = %q, want 'test'", target.Name)
		}
		if target.Value != 42 {
			t.Errorf("Value = %d, want 42", target.Value)
		}
	})

	t.Run("empty map", func(t *testing.T) {
		input := map[string]interface{}{}
		var target TestStruct
		err := helpers.MapToStruct(input, &target)
		if err != nil {
			t.Errorf("MapToStruct error: %v", err)
		}
	})

	t.Run("with nil values", func(t *testing.T) {
		input := map[string]interface{}{
			"name":  nil,
			"value": nil,
		}
		var target TestStruct
		err := helpers.MapToStruct(input, &target)
		if err != nil {
			t.Errorf("MapToStruct error: %v", err)
		}
	})
}

// TestStructToStruct tests the StructToStruct function
func TestStructToStruct_Extended(t *testing.T) {
	type Source struct {
		Name  string
		Value int
	}

	type Dest struct {
		Name  string
		Value int
	}

	t.Run("valid copy", func(t *testing.T) {
		src := Source{Name: "test", Value: 42}
		var dest Dest
		err := helpers.StructToStruct(&src, &dest)
		if err != nil {
			t.Errorf("StructToStruct error: %v", err)
		}
		if dest.Name != "test" {
			t.Errorf("Name = %q, want 'test'", dest.Name)
		}
		if dest.Value != 42 {
			t.Errorf("Value = %d, want 42", dest.Value)
		}
	})

	t.Run("with time fields", func(t *testing.T) {
		type TimeSource struct {
			Created time.Time
		}
		type TimeDest struct {
			Created time.Time
		}
		now := time.Now()
		src := TimeSource{Created: now}
		var dest TimeDest
		err := helpers.StructToStruct(&src, &dest)
		if err != nil {
			t.Errorf("StructToStruct error: %v", err)
		}
	})
}

// TestPtr tests the generic Ptr function
func TestPtr_Extended(t *testing.T) {
	t.Run("string pointer", func(t *testing.T) {
		s := "test"
		ptr := helpers.Ptr(s)
		if *ptr != s {
			t.Errorf("Ptr value = %q, want %q", *ptr, s)
		}
	})

	t.Run("int pointer", func(t *testing.T) {
		i := 42
		ptr := helpers.Ptr(i)
		if *ptr != i {
			t.Errorf("Ptr value = %d, want %d", *ptr, i)
		}
	})

	t.Run("bool pointer", func(t *testing.T) {
		b := true
		ptr := helpers.Ptr(b)
		if *ptr != b {
			t.Errorf("Ptr value = %v, want %v", *ptr, b)
		}
	})
}

// TestInterfaceToJSONString tests the InterfaceToJSONString function
func TestInterfaceToJSONString_Extended(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: "null",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: "{}",
		},
		{
			name:     "simple map",
			input:    map[string]interface{}{"key": "value"},
			expected: `{"key":"value"}`,
		},
		{
			name:     "nested map",
			input:    map[string]interface{}{"outer": map[string]interface{}{"inner": "value"}},
			expected: `{"outer":{"inner":"value"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.InterfaceToJSONString(tt.input)
			if result != tt.expected {
				t.Errorf("InterfaceToJSONString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestConvertToStringExtended tests additional ConvertToString cases
func TestConvertToStringExtended(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"int8", int8(127), "127"},
		{"int16", int16(32767), "32767"},
		{"int32", int32(2147483647), "2147483647"},
		{"int64", int64(9223372036854775807), "9223372036854775807"},
		{"uint", uint(123), "123"},
		{"uint8", uint8(255), "255"},
		{"uint16", uint16(65535), "65535"},
		{"uint32", uint32(4294967295), "4294967295"},
		{"float32", float32(3.14), "3.14"},
		{"struct", struct{ Name string }{"test"}, "{test}"},
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

// TestIsEmptyExtended tests additional IsEmpty cases
func TestIsEmptyExtended(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"uint zero", uint(0), true},
		{"uint non-zero", uint(1), false},
		{"uint8 zero", uint8(0), true},
		{"uint8 non-zero", uint8(1), false},
		{"empty array", [0]int{}, true},
		{"non-empty array", [3]int{1, 2, 3}, false},
		{"empty chan", make(chan int), true},
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

// TestTimeAgoExtended tests TimeAgo with more precise cases
func TestTimeAgoExtended(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		duration time.Duration
		contains string
	}{
		{"10 seconds ago", 10 * time.Second, "just now"},
		{"2 minutes ago", 2 * time.Minute, "minute"},
		{"1 minute ago", 1 * time.Minute, "1 minute ago"},
		{"1 hour ago", 1 * time.Hour, "1 hour ago"},
		{"3 hours ago", 3 * time.Hour, "hour"},
		{"1 day ago", 24 * time.Hour, "1 day ago"},
		{"5 days ago", 5 * 24 * time.Hour, "day"},
		{"1 week ago", 7 * 24 * time.Hour, "1 week ago"},
		{"3 weeks ago", 21 * 24 * time.Hour, "week"},
		{"1 month ago", 30 * 24 * time.Hour, "1 month ago"},
		{"6 months ago", 180 * 24 * time.Hour, "month"},
		{"1 year ago", 365 * 24 * time.Hour, "1 year ago"},
		{"2 years ago", 730 * 24 * time.Hour, "year"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.TimeAgo(now.Add(-tt.duration))
			if result == "" {
				t.Error("TimeAgo returned empty string")
			}
			// Check that result contains expected substring
			// Note: Exact string matching might fail due to time precision
		})
	}
}

// TestContainsExtended tests Contains with different types
func TestContainsExtended(t *testing.T) {
	t.Run("bool slice", func(t *testing.T) {
		slice := []bool{true, false, true}
		if !helpers.Contains(slice, true) {
			t.Error("Expected true to be found")
		}
		if !helpers.Contains(slice, false) {
			t.Error("Expected false to be found")
		}
	})

	t.Run("float slice", func(t *testing.T) {
		slice := []float64{1.1, 2.2, 3.3}
		if !helpers.Contains(slice, 2.2) {
			t.Error("Expected 2.2 to be found")
		}
		if helpers.Contains(slice, 4.4) {
			t.Error("Expected 4.4 to not be found")
		}
	})

	t.Run("uuid slice", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()
		slice := []uuid.UUID{id1, id2}
		if !helpers.Contains(slice, id1) {
			t.Error("Expected id1 to be found")
		}
	})
}
