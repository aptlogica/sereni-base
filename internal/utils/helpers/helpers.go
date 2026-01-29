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

// ConvertToInt converts string to int with error handling
func ConvertToInt(value string) (int, error) {
	return strconv.Atoi(value)
}

// ConvertToFloat converts string to float64 with error handling
func ConvertToFloat(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// ConvertToBool converts string to bool with error handling
func ConvertToBool(value string) (bool, error) {
	return strconv.ParseBool(value)
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

// Contains checks if a slice contains a specific element
func Contains(slice interface{}, element interface{}) bool {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
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
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
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

// SliceToString converts a slice of any type to a comma-separated string
func SliceToString(slice interface{}) string {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return ""
	}

	var parts []string
	for i := 0; i < s.Len(); i++ {
		parts = append(parts, ConvertToString(s.Index(i).Interface()))
	}

	return strings.Join(parts, ", ")
}

// StringToSlice converts a comma-separated string to a slice of strings
func StringToSlice(str string) []string {
	return SplitAndTrim(str, ",")
}

// MapKeys returns the keys of a map as a slicem reflect.Values
func extractMapInterfaces(values []reflect.Value, extractor func(reflect.Value) interface{}) []interface{} {
	result := make([]interface{}, len(values))
	for i, val := range values {
		result[i] = extractor(val)
	}
	return result
}

// MapKeys returns the keys of a map as a slice
func MapKeys(m interface{}) []interface{} {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		return nil
	}
	return extractMapInterfaces(v.MapKeys(), func(key reflect.Value) interface{} {
		return key.Interface()
	})
}

// MapValues returns the values of a map as a slice
func MapValues(m interface{}) []interface{} {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		return nil
	}
	return extractMapInterfaces(v.MapKeys(), func(key reflect.Value) interface{} {
		return v.MapIndex(key).Interface()
	})
}

// Reverse reverses a slice in place
func Reverse(slice interface{}) {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
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
	// debug print (keeps your original print from the snippet)
	fmt.Printf("type: %T, value: %#v\n", input["created_time"], input["created_time"])

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

// getTimeConverters returns all time-related type converters
func getTimeConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		// time.Time -> time.Time (pass through)
		{
			SrcType: time.Time{},
			DstType: time.Time{},
			Fn:      timeToTime,
		},
		// *time.Time -> time.Time (dereference)
		{
			SrcType: (*time.Time)(nil),
			DstType: time.Time{},
			Fn:      pointerTimeToTime,
		},
		// time.Time -> *time.Time (address)
		{
			SrcType: time.Time{},
			DstType: (*time.Time)(nil),
			Fn:      timeToPointerTime,
		},
		// *time.Time -> *time.Time (pass through)
		{
			SrcType: (*time.Time)(nil),
			DstType: (*time.Time)(nil),
			Fn:      pointerTimeToPointerTime,
		},
		// *time.Time -> *string (convert date to string pointer in yyyy-mm-dd format)
		{
			SrcType: (*time.Time)(nil),
			DstType: (*string)(nil),
			Fn:      pointerTimeToPointerString,
		},
	}
}

// timeToTime converts time.Time to time.Time (pass through)
func timeToTime(src interface{}) (interface{}, error) {
	return src, nil
}

// pointerTimeToTime converts *time.Time to time.Time (dereference)
func pointerTimeToTime(src interface{}) (interface{}, error) {
	if rt, ok := src.(*time.Time); ok && rt != nil {
		return *rt, nil
	}
	return time.Time{}, nil
}

// timeToPointerTime converts time.Time to *time.Time (address)
func timeToPointerTime(src interface{}) (interface{}, error) {
	if t, ok := src.(time.Time); ok {
		return &t, nil
	}
	return nil, nil
}

// pointerTimeToPointerTime converts *time.Time to *time.Time (pass through)
func pointerTimeToPointerTime(src interface{}) (interface{}, error) {
	return src, nil
}

// pointerTimeToPointerString converts *time.Time to *string (yyyy-mm-dd format)
func pointerTimeToPointerString(src interface{}) (interface{}, error) {
	if t, ok := src.(*time.Time); ok && t != nil {
		str := t.Format("2006-01-02")
		return &str, nil
	}
	return nil, nil
}

// getUUIDConverters returns all UUID-related type converters
func getUUIDConverters() []copier.TypeConverter {
	uuidToString := func(src interface{}) (interface{}, error) {
		if v, ok := src.(uuid.UUID); ok {
			return v.String(), nil
		}
		return src, nil
	}
	stringToUUID := func(src interface{}) (interface{}, error) {
		if s, ok := src.(string); ok {
			return uuid.Parse(s)
		}
		return src, nil
	}
	return []copier.TypeConverter{
		{SrcType: uuid.UUID{}, DstType: "", Fn: uuidToString},
		{SrcType: "", DstType: uuid.UUID{}, Fn: stringToUUID},
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

func StringPtr(s string) *string {
	return &s
}

func Float64Ptr(f float64) *float64 {
	return &f
}

func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func BoolPtr(b bool) *bool {
	return &b
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

// JoinStrings joins a slice of strings with a delimiter
func JoinStrings(parts []string, delimiter string) string {
	return strings.Join(parts, delimiter)
}

func ContainsString(slice []string, element string) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}
