package helpers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	// app_errors "serenibase/internal/app-errors"
	"strconv"
	"strings"
	"time"

	// "github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"

	"github.com/jinzhu/copier"
)

// GenerateID generates a random ID string
func GenerateID(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ConvertToString converts various types to string
func ConvertToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return strconv.FormatBool(v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// IsEmpty checks if a value is empty
func IsEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}

	return false
}

// validateSlice checks if the input is a valid slice and returns its reflect.Value
func validateSlice(slice interface{}) (reflect.Value, bool) {
	s := reflect.ValueOf(slice)
	return s, s.Kind() == reflect.Slice
}

// Contains checks if a slice contains a specific element
func Contains(slice interface{}, element interface{}) bool {
	s, ok := validateSlice(slice)
	if !ok {
		return false
	}

	for i := 0; i < s.Len(); i++ {
		if reflect.DeepEqual(s.Index(i).Interface(), element) {
			return true
		}
	}

	return false
}

// RemoveDuplicates removes duplicate elements from a slice
func RemoveDuplicates(slice interface{}) interface{} {
	s, ok := validateSlice(slice)
	if !ok {
		return slice
	}

	seen := make(map[interface{}]bool)
	result := reflect.MakeSlice(s.Type(), 0, s.Len())

	for i := 0; i < s.Len(); i++ {
		val := s.Index(i).Interface()
		if !seen[val] {
			seen[val] = true
			result = reflect.Append(result, s.Index(i))
		}
	}

	return result.Interface()
}

// TruncateString truncates a string to a specified length
func TruncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}

	if length <= 3 {
		return str[:length]
	}

	return str[:length-3] + "..."
}

// FormatFileSize formats a file size in bytes to human readable format
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// StringToSlice converts a comma-separated string to a slice of strings
func StringToSlice(str string) []string {
	return SplitAndTrim(str, ",")
}

// extractMapData is a generic helper to extract data from a map using reflection
func extractMapData(m interface{}, extractor func(reflect.Value, reflect.Value) interface{}) []interface{} {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		return nil
	}

	keys := v.MapKeys()
	result := make([]interface{}, len(keys))
	for i, key := range keys {
		result[i] = extractor(v, key)
	}
	return result
}

// MapKeys returns the keys of a map as a slice
func MapKeys(m interface{}) []interface{} {
	return extractMapData(m, func(_ reflect.Value, key reflect.Value) interface{} {
		return key.Interface()
	})
}

// MapValues returns the values of a map as a slice
func MapValues(m interface{}) []interface{} {
	return extractMapData(m, func(v reflect.Value, key reflect.Value) interface{} {
		return v.MapIndex(key).Interface()
	})
}

// SliceToString converts a slice of any type to a comma-separated string
func SliceToString(slice interface{}) string {
	s, ok := validateSlice(slice)
	if !ok {
		return ""
	}

	var parts []string
	for i := 0; i < s.Len(); i++ {
		parts = append(parts, ConvertToString(s.Index(i).Interface()))
	}

	return strings.Join(parts, ", ")
}

// Reverse reverses a slice in place
func Reverse(slice interface{}) {
	s, ok := validateSlice(slice)
	if !ok {
		return
	}

	for i, j := 0, s.Len()-1; i < j; i, j = i+1, j-1 {
		vi, vj := s.Index(i), s.Index(j)
		temp := vi.Interface()
		vi.Set(vj)
		vj.Set(reflect.ValueOf(temp))
	}
}

// formatTimeUnit formats a time unit with singular/plural handling
func formatTimeUnit(count int, unit string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s ago", unit)
	}
	return fmt.Sprintf("%d %ss ago", count, unit)
}

// TimeAgo returns a human-readable time difference
func TimeAgo(t time.Time) string {
	diff := time.Now().Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return formatTimeUnit(int(diff.Minutes()), "minute")
	case diff < 24*time.Hour:
		return formatTimeUnit(int(diff.Hours()), "hour")
	case diff < 7*24*time.Hour:
		return formatTimeUnit(int(diff.Hours()/24), "day")
	case diff < 30*24*time.Hour:
		return formatTimeUnit(int(diff.Hours()/(24*7)), "week")
	case diff < 365*24*time.Hour:
		return formatTimeUnit(int(diff.Hours()/(24*30)), "month")
	default:
		return formatTimeUnit(int(diff.Hours()/(24*365)), "year")
	}
}

func MapToStruct(input map[string]interface{}, target interface{}) error {
	// JSON round-trip: map -> JSON -> struct
	// This relies on encoding/json behavior:
	//  - strings with RFC3339 parse into time.Time
	//  - uuid.UUID implements encoding.TextUnmarshaler so JSON strings parse into uuid.UUID
	b, err := json.Marshal(input)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, target); err != nil {
		return err
	}
	return nil
}

// createConverter creates a type converter with the given source type, destination type, and conversion function
func createConverter(srcType, dstType interface{}, fn func(interface{}) (interface{}, error)) copier.TypeConverter {
	return copier.TypeConverter{
		SrcType: srcType,
		DstType: dstType,
		Fn:      fn,
	}
}

// getTimeConverters returns all time-related type converters
func getTimeConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		// time.Time <-> time.Time (pass through)
		createConverter(time.Time{}, time.Time{}, passThrough),
		// *time.Time -> time.Time (dereference)
		createConverter((*time.Time)(nil), time.Time{}, func(src interface{}) (interface{}, error) {
			if rt, ok := src.(*time.Time); ok && rt != nil {
				return *rt, nil
			}
			return time.Time{}, nil
		}),
		// time.Time -> *time.Time (address)
		createConverter(time.Time{}, (*time.Time)(nil), func(src interface{}) (interface{}, error) {
			if t, ok := src.(time.Time); ok {
				return &t, nil
			}
			return nil, nil
		}),
		// *time.Time <-> *time.Time (pass through)
		createConverter((*time.Time)(nil), (*time.Time)(nil), passThrough),
		// *time.Time -> *string (convert date to string pointer in yyyy-mm-dd format)
		createConverter((*time.Time)(nil), (*string)(nil), func(src interface{}) (interface{}, error) {
			if t, ok := src.(*time.Time); ok && t != nil {
				str := t.Format("2006-01-02")
				return &str, nil
			}
			return nil, nil
		}),
	}
}

// passThrough is a generic pass-through converter (returns src as-is)
func passThrough(src interface{}) (interface{}, error) {
	return src, nil
}

// getUUIDConverters returns all UUID-related type converters
func getUUIDConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		createConverter(uuid.UUID{}, "", func(src interface{}) (interface{}, error) {
			if v, ok := src.(uuid.UUID); ok {
				return v.String(), nil
			}
			return src, nil
		}),
		createConverter("", uuid.UUID{}, func(src interface{}) (interface{}, error) {
			if s, ok := src.(string); ok {
				return uuid.Parse(s)
			}
			return src, nil
		}),
	}
}

// StructToStruct: copies from src -> dest using jinzhu/copier with converters set up
// to handle time.Time <-> *time.Time, *time.Time -> *string and uuid.UUID <-> string conversions.
func StructToStruct(src, dest interface{}) error {
	// Prepare converters for copier.Option
	converters := append(getTimeConverters(), getUUIDConverters()...)

	// Option: set DeepCopy true if you want deep copy semantics
	opt := copier.Option{
		DeepCopy:   true,
		Converters: converters,
	}

	// copier expects pointers for destination in most cases
	// copy with option
	if err := copier.CopyWithOption(dest, src, opt); err != nil {
		// note: some older copier versions swallow converter errors (see issues),
		// so if you rely on converter error propagation, check your copier version.
		return err
	}

	// Extra safety: if dest is pointer-to-pointer/time mismatches, we can try
	// a fallback attempt using reflection to set fields individually (not included here).
	return nil
}

// Ptr returns a pointer to the given value (generic pointer helper)
func Ptr[T any](v T) *T {
	return &v
}

// StringPtr returns a pointer to the given string (kept for backward compatibility)
func StringPtr(s string) *string {
	return Ptr(s)
}

// Float64Ptr returns a pointer to the given float64 (kept for backward compatibility)
func Float64Ptr(f float64) *float64 {
	return Ptr(f)
}

// BoolPtr returns a pointer to the given bool (kept for backward compatibility)
func BoolPtr(b bool) *bool {
	return Ptr(b)
}

func ToSnakeCase(str string) string {
	reg := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := reg.ReplaceAllString(str, "${1}_${2}")

	// Replace spaces and hyphens with underscores
	snake = strings.ReplaceAll(snake, " ", "_")
	snake = strings.ReplaceAll(snake, "-", "_")

	// Convert to lowercase
	return strings.ToLower(snake)
}

func InterfaceToJSONString(val map[string]interface{}) string {
	if val == nil {
		return "null"
	}
	bytes, err := json.Marshal(val)
	if err != nil {
		return "null"
	}
	return string(bytes)
}

// splitAndProcess splits string and applies transformation
func splitAndProcess(str string, delimiter string, keepEmpty bool) []string {
	if str == "" {
		return []string{}
	}
	parts := strings.Split(str, delimiter)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if keepEmpty || trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// SplitAndTrim splits a string by delimiter and trims whitespace from each part
func SplitAndTrim(str string, delimiter string) []string {
	return splitAndProcess(str, delimiter, false)
}
