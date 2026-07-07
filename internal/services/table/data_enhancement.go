// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
	"github.com/google/uuid"
)

type columnSplitStrategy struct {
	kind      string
	separator string
	action    string
	value     int
	pattern   string
	regex     *regexp.Regexp
}

const (
	removeFormattingRowSkipped removeFormattingRowStatus = iota
	removeFormattingRowUpdated
	errFixedLengthPositive = "fixed_length value must be greater than zero"
	errFixedLengthInvalid  = "fixed_length value is invalid"
)

func quoteIdentifier(identifier string) string {
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(identifier, "\"", "\"\""))
}

func formatQualifiedTable(schemaName, tableName string) string {
	return fmt.Sprintf("%s.%s", quoteIdentifier(schemaName), quoteIdentifier(tableName))
}

var (
	symbolCharSet = map[rune]struct{}{
		'@': {}, '#': {}, '$': {}, '%': {}, '^': {}, '&': {}, '*': {},
		'!': {}, '~': {}, '`': {}, '|': {}, '\\': {},
	}
	currencyCharSet = map[rune]struct{}{
		'₹': {}, '$': {}, '€': {}, '£': {}, '¥': {},
	}
	bracketCharSet = map[rune]struct{}{
		'(': {}, ')': {}, '[': {}, ']': {}, '{': {}, '}': {}, '<': {}, '>': {},
	}
	punctuationCharSet = map[rune]struct{}{
		'.': {}, ',': {}, ';': {}, ':': {}, '!': {}, '?': {}, '\'': {}, '"': {}, '-': {},
	}
	// Collapse only runs of whitespace that occur between two non-space characters
	multiSpaceBetweenWordsRegex = regexp.MustCompile(`(\S)\s{2,}(\S)`)
	currencySymbolRegex         = regexp.MustCompile(`[₹$€£¥]`)
	numericSeparatorRegex       = regexp.MustCompile(`,`)
	flexibleDateLayouts         = []string{
		time.RFC3339Nano, time.RFC3339,
		"2006-01-02 15:04:05", "2006-01-02 15:04:05.999999", "2006-01-02 15:04",
		"2006-01-02", "20060102", "02-01-2006", "02/01/2006", "02.01.2006",
		"02 01 2006",
		"01-02-2006", "01/02/2006", "01.02.2006",
		"2006/01/02", "2006.01.02",
		"02 Jan 2006", "02 January 2006", "Jan 02 2006", "January 02 2006",
	}
	regexEmail    = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	regexURL      = regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s<>()]+`)
	regexHashtag  = regexp.MustCompile(`(?:^|\s)(#[A-Za-z0-9_]+)`)
	regexMention  = regexp.MustCompile(`(?:^|\s)(@[A-Za-z0-9_.-]+)`)
	regexKeywords = regexp.MustCompile(`[\p{L}\p{N}]+`)
	regexEmoji    = regexp.MustCompile(`(?:[\x{1F1E6}-\x{1F1FF}]{2}|[#*0-9]\x{FE0F}?\x{20E3}|[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{27BF}\x{2934}-\x{2935}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F000}-\x{1FAFF}](?:\x{FE0F}|\x{FE0E})?(?:\x{200D}[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{27BF}\x{2934}-\x{2935}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F000}-\x{1FAFF}](?:\x{FE0F}|\x{FE0E})?)*)`)
	regexPhone    = regexp.MustCompile(`(?:^|[^\d])(\+?\d[\d\s().-]{7,}\d)(?:$|[^\d])`)
)

// --- extraction helper functions ---

func CleanExtractionMatch(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), ".,;:!?)")
}

func ExtractFirstEmail(s string) (string, bool) {
	m := CleanExtractionMatch(regexEmail.FindString(s))
	if m != "" {
		return m, true
	}
	return "", false
}

func ExtractFirstURL(s string) (string, bool) {
	m := CleanExtractionMatch(regexURL.FindString(s))
	if m != "" {
		return m, true
	}
	return "", false
}

func ExtractURLsFromText(s string) (string, bool) {
	matches := regexURL.FindAllString(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		cleaned := CleanExtractionMatch(m)
		if cleaned == "" {
			continue
		}
		out = append(out, cleaned)
	}
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, ", "), true
}

func ExtractDomainFromText(s string) (string, bool) {
	if email, ok := ExtractFirstEmail(s); ok {
		parts := strings.SplitN(email, "@", 2)
		if len(parts) == 2 {
			domain := strings.TrimSpace(parts[1])
			domain = strings.Trim(domain, ",;:)}]")
			domain = strings.TrimPrefix(domain, "www.")
			return domain, true
		}
	}
	if uStr, ok := ExtractFirstURL(s); ok {
		if u, err := url.Parse(uStr); err == nil {
			host := u.Hostname()
			host = strings.TrimPrefix(host, "www.")
			return host, true
		}
	}
	return "", false
}

func ExtractHashtagsFromText(s string) (string, bool) {
	matches := regexHashtag.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(m[1])
		if raw == "" {
			continue
		}
		out = append(out, raw)
	}
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, ", "), true
}

func ExtractMentionsFromText(s string) (string, bool) {
	matches := regexMention.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(m[1])
		if raw == "" {
			continue
		}
		out = append(out, raw)
	}
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, ", "), true
}

func ExtractKeywordsFromText(s string) (string, bool) {
	matches := regexKeywords.FindAllString(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	stopWords := map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "the": {}, "or": {}, "but": {}, "to": {}, "of": {}, "in": {}, "on": {}, "at": {}, "for": {}, "with": {}, "from": {}, "by": {},
		"is": {}, "are": {}, "was": {}, "were": {}, "be": {}, "been": {}, "it": {}, "this": {}, "that": {}, "these": {}, "those": {}, "as": {}, "into": {},
		"over": {}, "under": {}, "about": {}, "after": {}, "before": {}, "between": {}, "through": {}, "during": {}, "without": {}, "within": {},
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, len(matches))
	for _, tok := range matches {
		t := strings.TrimSpace(tok)
		if t == "" {
			continue
		}
		tl := strings.ToLower(t)
		if _, ok := stopWords[tl]; ok {
			continue
		}
		if len([]rune(t)) <= 2 {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if len(out) == 0 {
		return "", false
	}
	if len(out) > 20 {
		out = out[:20]
	}
	return strings.Join(out, ", "), true
}

func ExtractEmojiFromText(s string) (string, bool) {
	out := regexEmoji.FindAllString(s, -1)
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, ", "), true
}

func ExtractPhoneNumberFromText(s string) (string, bool) {
	m := regexPhone.FindStringSubmatch(s)
	if len(m) < 2 {
		return "", false
	}
	phone := strings.TrimSpace(m[1])
	if phone == "" {
		return "", false
	}
	return phone, true
}

func ExtractEmailPrefixFromText(s string) (string, bool) {
	matches := regexEmail.FindAllString(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	prefixes := make([]string, 0, len(matches))
	for _, email := range matches {
		parts := strings.SplitN(strings.TrimSpace(email), "@", 2)
		if len(parts) < 2 {
			continue
		}
		prefix := strings.TrimSpace(parts[0])
		if prefix == "" {
			continue
		}
		prefixes = append(prefixes, prefix)
	}
	if len(prefixes) == 0 {
		return "", false
	}
	return strings.Join(prefixes, ", "), true
}

func ExtractBetweenCharactersFromText(s, startAfter, endBefore string) (string, bool) {
	if startAfter == "" || endBefore == "" {
		return "", false
	}
	startIdx := strings.Index(s, startAfter)
	if startIdx == -1 {
		return "", false
	}
	startPos := startIdx + len(startAfter)
	if startPos >= len(s) {
		return "", false
	}
	rest := s[startPos:]
	endIdx := strings.Index(rest, endBefore)
	if endIdx == -1 {
		return "", false
	}
	extracted := strings.TrimSpace(rest[:endIdx])
	if extracted == "" {
		return "", false
	}
	return extracted, true
}

func (s tableManagementService) GetSelectedColumnsFromRequest(columnsData []dto.ColumnResponse, requested []string) ([]string, error) {
	columnSet := make(map[string]string, len(columnsData))
	for _, col := range columnsData {
		columnSet[col.ID.String()] = col.ColumnName
	}

	selectedColumns := make([]string, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, colID := range requested {
		columnID := strings.TrimSpace(colID)
		if columnID == "" {
			continue
		}
		if _, ok := seen[columnID]; ok {
			continue
		}
		columnName, exists := columnSet[columnID]
		if !exists {
			return nil, app_errors.ColumnNotFound
		}
		seen[columnID] = struct{}{}
		selectedColumns = append(selectedColumns, columnName)
	}
	if len(selectedColumns) == 0 {
		return nil, app_errors.InvalidPayload
	}
	return selectedColumns, nil
}

func CleanWhitespaceValue(value, trimMode string) string {
	cleaned := value
	switch trimMode {
	case "trim_both":
		cleaned = strings.TrimSpace(cleaned)
	case "trim_leading":
		cleaned = strings.TrimLeftFunc(cleaned, unicode.IsSpace)
	case "trim_trailing":
		cleaned = strings.TrimRightFunc(cleaned, unicode.IsSpace)
	case "collapse_spaces":
		cleaned = CollapseInternalSpaces(cleaned)
	default:
		cleaned = strings.TrimSpace(cleaned)
	}

	return cleaned
}

func CollapseInternalSpaces(s string) string {
	return multiSpaceBetweenWordsRegex.ReplaceAllString(s, "$1 $2")
}

func NormalizeValue(value, caseFormat string) string {
	switch caseFormat {
	case "lowercase":
		return strings.ToLower(value)
	case "uppercase":
		return strings.ToUpper(value)
	case "title_case":
		return ToTitleCase(value)
	case "sentence_case":
		return ToSentenceCase(value)
	default:
		return value
	}
}

func ToTitleCase(s string) string {
	var b strings.Builder
	prevIsLetter := false
	for _, r := range s {
		if !prevIsLetter && unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
			prevIsLetter = true
			continue
		}
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToLower(r))
			prevIsLetter = true
			continue
		}
		b.WriteRune(r)
		prevIsLetter = false
	}
	return b.String()
}

func ToSentenceCase(s string) string {
	var b strings.Builder
	startOfSentence := true
	for _, r := range s {
		if startOfSentence && unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
			startOfSentence = false
			continue
		}
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			startOfSentence = true
		}
	}
	return b.String()
}

func (s tableManagementService) ComputeFindReplace(strValue, findValue, replaceValue, matchType string, ignoreRe *regexp.Regexp) (bool, string) {
	switch matchType {
	case "match_case":
		if strings.Contains(strValue, findValue) {
			return true, strings.ReplaceAll(strValue, findValue, replaceValue)
		}
	case "ignore_case":
		if ignoreRe != nil && ignoreRe.MatchString(strValue) {
			return true, ignoreRe.ReplaceAllString(strValue, replaceValue)
		}
	case "match_entire_value":
		if strValue == findValue {
			return true, replaceValue
		}
	}
	return false, ""
}

func RemoveCharSetForType(removeType string) map[rune]struct{} {
	switch removeType {
	case "symbols":
		return symbolCharSet
	case "currency_symbols":
		return currencyCharSet
	case "brackets":
		return bracketCharSet
	case "punctuation":
		return punctuationCharSet
	default:
		return nil
	}
}

func ComputeRemoveSpecialCharacters(strValue, removeType string, customChars []string) (bool, string) {
	if removeType == "custom" {
		newVal := strValue
		matched := false
		for _, ch := range customChars {
			if strings.Contains(newVal, ch) {
				matched = true
				newVal = strings.ReplaceAll(newVal, ch, "")
			}
		}
		if !matched {
			return false, ""
		}
		return true, newVal
	}

	removeSet := RemoveCharSetForType(removeType)
	if removeSet == nil {
		return false, ""
	}
	var b strings.Builder
	b.Grow(len(strValue))
	matched := false
	for _, r := range strValue {
		if _, ok := removeSet[r]; ok {
			matched = true
			continue
		}
		b.WriteRune(r)
	}
	if !matched {
		return false, ""
	}
	return true, b.String()
}

func StripFormattingByType(value string, formatting string, customPatterns []string) (bool, string, bool) {
	switch formatting {
	case "currency":
		return RemoveCurrencyFormatting(value)
	case "percentage":
		return RemovePercentageFormatting(value)
	case "separator":
		return RemoveSeparatorFormatting(value)
	case "phone":
		return RemovePhoneFormatting(value)
	case "date":
		return NormalizeDateFormatting(value)
	case "custom":
		return RemoveCustomFormatting(value, customPatterns)
	default:
		return false, "", false
	}
}

func RemoveCurrencyFormatting(value string) (bool, string, bool) {
	replaced := currencySymbolRegex.ReplaceAllString(value, "")
	replaced = numericSeparatorRegex.ReplaceAllString(replaced, "")
	return replaced != value, replaced, false
}

func RemovePercentageFormatting(value string) (bool, string, bool) {
	replaced := strings.ReplaceAll(value, "%", "")
	return replaced != value, replaced, false
}

func RemoveSeparatorFormatting(value string) (bool, string, bool) {
	replaced := numericSeparatorRegex.ReplaceAllString(value, "")
	return replaced != value, replaced, false
}

func RemovePhoneFormatting(value string) (bool, string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false, "", false
	}
	if _, ok := ParseFlexibleDate(trimmed); ok {
		return false, "", false
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	changed := false
	plusAllowed := true
	for _, r := range trimmed {
		switch {
		case (r >= '0' && r <= '9') || (r == '+' && plusAllowed):
			b.WriteRune(r)
			plusAllowed = false
		default:
			changed = true
		}
	}
	replaced := b.String()
	if replaced == trimmed {
		return false, "", false
	}
	return changed || replaced != value, replaced, false
}

func RemoveCustomFormatting(value string, customPatterns []string) (bool, string, bool) {
	if len(customPatterns) == 0 {
		return false, "", false
	}
	replaced := value
	for _, p := range customPatterns {
		if p == "" {
			continue
		}
		replaced = strings.ReplaceAll(replaced, p, "")
	}
	if replaced == value {
		return false, "", false
	}
	return true, replaced, false
}

func NormalizeDateFormatting(value string) (bool, string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false, "", false
	}
	if parsed, ok := ParseFlexibleDate(trimmed); ok {
		return true, parsed.Format(dateOutputLayout), false
	}
	return false, "", false
}

func ParseFlexibleDate(value string) (time.Time, bool) {
	for _, layout := range flexibleDateLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func (s tableManagementService) ProcessRemoveFormattingCell(
	rowID interface{},
	columnName string,
	value interface{},
	formatting string,
	customPatterns []string,
) (*dto.UpdateColumnValueRequest, bool) {
	formatted := ToStringValue(value)

	changed, newValue, failed := StripFormattingByType(formatted, formatting, customPatterns)
	if failed {
		return nil, false
	}
	if !changed {
		return nil, false
	}

	if strings.EqualFold(strings.TrimSpace(formatting), "date") {
		if parsed, ok := ParseFlexibleDate(formatted); ok {
			return &dto.UpdateColumnValueRequest{
				Id:     rowID,
				Column: columnName,
				Value:  parsed.Format(dateOutputLayout),
			}, true
		}
		return nil, false
	}

	updatedValue, ok := InferFormattedCellValue(newValue)
	if !ok {
		return nil, false
	}

	return &dto.UpdateColumnValueRequest{
		Id:     rowID,
		Column: columnName,
		Value:  updatedValue,
	}, true
}

func ToStringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	default:
		return fmt.Sprint(v)
	}
}

func InferFormattedCellValue(value string) (interface{}, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value, true
	}
	if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return parsed, true
	}
	if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return parsed, true
	}
	if parsed, err := strconv.ParseBool(trimmed); err == nil {
		return parsed, true
	}
	return trimmed, true
}

func LookupRowValue(row map[string]interface{}, columnName string) (interface{}, bool) {
	if value, exists := row[columnName]; exists {
		return value, true
	}
	normalizedTarget := strings.ToLower(strings.TrimSpace(columnName))
	for key, value := range row {
		if strings.ToLower(strings.TrimSpace(key)) == normalizedTarget {
			return value, true
		}
	}
	return nil, false
}

func (s tableManagementService) RemoveFormatting(ctx context.Context, schemaName string, req dto.RemoveFormattingRequest) (dto.RemoveFormattingResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveFormattingResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveFormattingResponse{}, err
	}

	selectedColumns, err := s.GetSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.RemoveFormattingResponse{}, err
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	limit := columnActionBatchSize
	offset := 0
	totalResult := dto.RemoveFormattingResponse{}

	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	for {
		params := dbModels.QueryParams{
			Select: selectColumns,
			Limit:  &limit,
			Offset: &offset,
		}
		rows, err := s.repo.TableService.GetTableData(tableName, params)
		if err != nil {
			return dto.RemoveFormattingResponse{}, app_errors.LogDatabaseError(err, "failed to fetch rows for remove formatting")
		}

		updates, result := s.BuildRemoveFormattingUpdates(rows, req.Formatting, req.CustomPattern, selectedColumns)

		if len(updates) > 0 {
			if err := s.ApplyRemoveFormattingUpdates(ctx, fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias), updates); err != nil {
				return dto.RemoveFormattingResponse{}, err
			}
		}

		totalResult.ScannedRecords += result.ScannedRecords
		totalResult.UpdatedRecords += result.UpdatedRecords
		totalResult.SkippedRecords += result.SkippedRecords
		totalResult.FailedRecords += result.FailedRecords

		lg.Info().
			Str("model_id", req.ModelID).
			Int("batch_offset", offset).
			Int("batch_rows", len(rows)).
			Int("batch_updates", len(updates)).
			Str("formatting", req.Formatting).
			Msg("RemoveFormatting batch processed")

		if len(rows) < limit {
			break
		}
		offset += limit
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", 0).
		Int("scanned_records", totalResult.ScannedRecords).
		Int("updated_records", totalResult.UpdatedRecords).
		Int("skipped_records", totalResult.SkippedRecords).
		Int("failed_records", totalResult.FailedRecords).
		Str("formatting", req.Formatting).
		Msg("Remove formatting action completed")

	return totalResult, nil
}

func (s tableManagementService) FetchTableRowsForTrim(ctx context.Context, tableName string, selectColumns []string) ([]map[string]interface{}, error) {
	rows, err := s.repo.TableService.GetTableData(tableName, dbModels.QueryParams{Select: selectColumns})
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch rows for trim whitespace")
	}
	return rows, nil
}

func (s tableManagementService) ValidateColumnsAllowed(ctx context.Context, schemaName string, modelID string, columnIDs []string) error {
	if len(columnIDs) == 0 {
		return nil
	}

	cols, err := s.GetColumnsByModelID(ctx, schemaName, modelID)
	if err != nil {
		return err
	}

	uidtByID := make(map[string]string, len(cols))
	for _, col := range cols {
		uidtByID[col.ID.String()] = col.UIDT
	}

	blocked := map[string]struct{}{
		"attachment":  {},
		"link":        {},
		"links":       {},
		"lookup":      {},
		"user":        {},
		"formula":     {},
		"rating":      {},
		"multiselect": {},
		"select":      {},
	}

	for _, id := range columnIDs {
		uidt, ok := uidtByID[id]
		if !ok {
			return app_errors.ColumnNotFound
		}
		if uidt == "" {
			continue
		}
		if _, blockedType := blocked[strings.ToLower(uidt)]; blockedType {
			return app_errors.UpdateNotAllowed
		}
	}

	return nil
}

func (s tableManagementService) ValidateColumnAllowedForSplit(ctx context.Context, schemaName string, modelID string, columnID string) error {
	cols, err := s.GetColumnsByModelID(ctx, schemaName, modelID)
	if err != nil {
		return err
	}

	allowed := map[string]struct{}{"text": {}, "longtext": {}, "url": {}, "email": {}}

	for _, col := range cols {
		if col.ID.String() == columnID {
			if col.UIDT == "" {
				return app_errors.SplitNotPossible
			}
			if _, ok := allowed[strings.ToLower(col.UIDT)]; !ok {
				return app_errors.SplitNotPossible
			}
			return nil
		}
	}

	return app_errors.ColumnNotFound
}

func (s tableManagementService) TrimWhitespace(ctx context.Context, schemaName string, req dto.TrimWhitespaceRequest) (dto.TrimWhitespaceResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.TrimWhitespaceResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.TrimWhitespaceResponse{}, err
	}

	selectedColumns, err := s.GetSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.TrimWhitespaceResponse{}, err
	}

	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rows, err := s.FetchTableRowsForTrim(ctx, tableName, selectColumns)
	if err != nil {
		return dto.TrimWhitespaceResponse{}, err
	}

	updates, result := s.BuildTrimUpdates(rows, selectedColumns, req.TrimMode)

	if len(updates) > 0 {
		if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, model.Alias, updates); err != nil {
			return dto.TrimWhitespaceResponse{}, err
		}
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int("total_scanned", result.TotalScanned).
		Int("total_updated", result.TotalUpdated).
		Int("total_skipped", result.TotalSkipped).
		Int("total_rows", result.TotalRows).
		Int("total_rows_updated", result.TotalRowsUpdated).
		Int("total_rows_skipped", result.TotalRowsSkipped).
		Msg("Trim whitespace action completed")

	return result, nil
}

func (s tableManagementService) BuildTrimUpdates(rows []map[string]interface{}, selectedColumns []string, trimMode string) ([]dto.UpdateColumnValueRequest, dto.TrimWhitespaceResponse) {
	result := dto.TrimWhitespaceResponse{
		TotalScanned: len(rows) * len(selectedColumns),
		TotalRows:    len(rows),
	}
	if len(rows) == 0 {
		return nil, result
	}

	updates := make([]dto.UpdateColumnValueRequest, 0)
	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalSkipped += len(selectedColumns)
			result.TotalRowsSkipped++
			continue
		}
		rowUpdates, skipped, rowUpdated := s.BuildTrimUpdatesForRow(rowID, row, selectedColumns, trimMode)
		if skipped > 0 {
			result.TotalSkipped += skipped
		}
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
			result.TotalUpdated += len(rowUpdates)
		}
		if rowUpdated {
			result.TotalRowsUpdated++
		} else {
			result.TotalRowsSkipped++
		}
	}
	return updates, result
}

// Exported wrappers for testing from external packages.
func BuildTrimUpdatesPublic(rows []map[string]interface{}, selectedColumns []string, trimMode string) ([]dto.UpdateColumnValueRequest, dto.TrimWhitespaceResponse) {
	var s tableManagementService
	return s.BuildTrimUpdates(rows, selectedColumns, trimMode)
}

func BuildTrimUpdatesForRowPublic(rowID interface{}, row map[string]interface{}, selectedColumns []string, trimMode string) ([]dto.UpdateColumnValueRequest, int, bool) {
	var s tableManagementService
	return s.BuildTrimUpdatesForRow(rowID, row, selectedColumns, trimMode)
}

func BuildCaseNormalizationUpdatesPublic(rows []map[string]interface{}, selectedColumns []string, caseFormat string) ([]dto.UpdateColumnValueRequest, dto.CaseNormalizationResponse) {
	var s tableManagementService
	return s.BuildCaseNormalizationUpdates(rows, selectedColumns, caseFormat)
}

func BuildCaseNormalizationUpdatesForRowPublic(rowID interface{}, row map[string]interface{}, selectedColumns []string, caseFormat string) ([]dto.UpdateColumnValueRequest, int, bool) {
	var s tableManagementService
	return s.BuildCaseNormalizationUpdatesForRow(rowID, row, selectedColumns, caseFormat)
}

func BuildFindReplaceUpdatesPublic(rows []map[string]interface{}, selectedColumns []string, findValue, replaceValue, matchType string, ignoreRe *regexp.Regexp) ([]dto.UpdateColumnValueRequest, dto.FindReplaceResponse) {
	var s tableManagementService
	return s.BuildFindReplaceUpdates(rows, selectedColumns, findValue, replaceValue, matchType, ignoreRe)
}

func BuildFindReplaceUpdatesForRowPublic(rowID interface{}, row map[string]interface{}, selectedColumns []string, findValue, replaceValue, matchType string, ignoreRe *regexp.Regexp) ([]dto.UpdateColumnValueRequest, int, bool, int, int) {
	var s tableManagementService
	return s.BuildFindReplaceUpdatesForRow(rowID, row, selectedColumns, findValue, replaceValue, matchType, ignoreRe)
}

func BuildRemoveFormattingUpdatesPublic(rows []map[string]interface{}, formatting string, customPatterns []string, selectedColumns []string) ([]dto.UpdateColumnValueRequest, dto.RemoveFormattingResponse) {
	var s tableManagementService
	return s.BuildRemoveFormattingUpdates(rows, formatting, customPatterns, selectedColumns)
}

func BuildRemoveFormattingUpdatesForRowPublic(row map[string]interface{}, formatting string, customPatterns []string, selectedColumns []string) ([]dto.UpdateColumnValueRequest, removeFormattingRowStatus) {
	var s tableManagementService
	return s.BuildRemoveFormattingUpdatesForRow(row, formatting, customPatterns, selectedColumns)
}

func BuildRemoveSpecialCharactersUpdatesPublic(rows []map[string]interface{}, selectedColumns []string, removeType string, customChars []string) ([]dto.UpdateColumnValueRequest, dto.RemoveSpecialCharactersResponse) {
	var s tableManagementService
	return s.BuildRemoveSpecialCharactersUpdates(rows, selectedColumns, removeType, customChars)
}

func BuildRemoveSpecialCharactersUpdatesForRowPublic(rowID interface{}, row map[string]interface{}, selectedColumns []string, removeType string, customChars []string) ([]dto.UpdateColumnValueRequest, int, bool, int, int) {
	var s tableManagementService
	return s.BuildRemoveSpecialCharactersUpdatesForRow(rowID, row, selectedColumns, removeType, customChars)
}

func DetermineMatchCasePublic(mode string) bool {
	var s tableManagementService
	return s.DetermineMatchCase(mode)
}

func BuildDeleteDuplicatesQueryPublic(tableName, partitionBy, orderBy, condition string) string {
	var s tableManagementService
	return s.BuildDeleteDuplicatesQuery(tableName, partitionBy, orderBy, condition)
}

func BuildUpdateDuplicatesQueryPublic(tableName, partitionBy, orderBy, condition, setClause string) string {
	var s tableManagementService
	return s.BuildUpdateDuplicatesQuery(tableName, partitionBy, orderBy, condition, setClause)
}

func BuildDuplicateKeyExpressionsPublic(selectedColumns []string, matchCase bool) (string, string) {
	var s tableManagementService
	return s.BuildDuplicateKeyExpressions(selectedColumns, matchCase)
}

func BuildDuplicateKeepOrderByPublic(keepRule string) string {
	var s tableManagementService
	return s.BuildDuplicateKeepOrderBy(keepRule)
}

func ParseColumnSplitStrategyPublic(splitBy dto.SplitByRequest) (columnSplitStrategy, error) {
	var s tableManagementService
	return s.ParseColumnSplitStrategy(splitBy)
}

func BuildSplitColumnNamesPublic(columns []dto.ColumnResponse, baseName string, count int) ([]string, error) {
	var s tableManagementService
	return s.BuildSplitColumnNames(columns, baseName, count)
}

func BuildSplitColumnTitlePublic(baseTitle, baseName string, index int) string {
	var s tableManagementService
	return s.BuildSplitColumnTitle(baseTitle, baseName, index)
}

// GetSelectedColumnsFromRequestPublic exposes GetSelectedColumnsFromRequest for unit tests.
func GetSelectedColumnsFromRequestPublic(columnsData []dto.ColumnResponse, requested []string) ([]string, error) {
	var s tableManagementService
	return s.GetSelectedColumnsFromRequest(columnsData, requested)
}

func ComputeSplitOrderIndexesPublic(columns []dto.ColumnResponse, selectedIndex int, where string, count int) ([]float64, error) {
	var s tableManagementService
	return s.ComputeSplitOrderIndexes(columns, selectedIndex, where, count)
}

func GetColumnByIDFromListPublic(columns []dto.ColumnResponse, id string) (dto.ColumnResponse, int, error) {
	var s tableManagementService
	return s.GetColumnByIDFromList(columns, id)
}

func ComputeFindReplacePublic(strValue, findValue, replaceValue, matchType string, ignoreRe *regexp.Regexp) (bool, string) {
	var s tableManagementService
	return s.ComputeFindReplace(strValue, findValue, replaceValue, matchType, ignoreRe)
}

func ProcessRemoveFormattingCellPublic(rowID interface{}, columnName string, value interface{}, formatting string, customPatterns []string) (*dto.UpdateColumnValueRequest, bool) {
	var s tableManagementService
	return s.ProcessRemoveFormattingCell(rowID, columnName, value, formatting, customPatterns)
}

func GetSplitSQLArrayExprPublic(columnName string, strategy columnSplitStrategy) (string, []interface{}) {
	var s tableManagementService
	return s.GetSplitSQLArrayExpr(columnName, strategy)
}

func BuildMergeUpdatesPublic(rows []map[string]interface{}, selectedColumns []string, sep, newColumnName string) ([]dto.UpdateColumnValueRequest, dto.MergeColumnsResponse) {
	var s tableManagementService
	return s.BuildMergeUpdates(rows, selectedColumns, sep, newColumnName)
}

func (s tableManagementService) BuildTrimUpdatesForRow(rowID interface{}, row map[string]interface{}, selectedColumns []string, trimMode string) ([]dto.UpdateColumnValueRequest, int, bool) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	skipped := 0
	rowUpdated := false
	for _, columnName := range selectedColumns {
		value, exists := row[columnName]
		if !exists || value == nil {
			skipped++
			continue
		}
		strValue, ok := value.(string)
		if !ok {
			skipped++
			continue
		}
		cleaned := CleanWhitespaceValue(strValue, trimMode)
		if cleaned == strValue {
			skipped++
			continue
		}
		updates = append(updates, dto.UpdateColumnValueRequest{Id: rowID, Column: columnName, Value: cleaned})
		rowUpdated = true
	}
	return updates, skipped, rowUpdated
}

func (s tableManagementService) BuildCaseNormalizationUpdates(rows []map[string]interface{}, selectedColumns []string, caseFormat string) ([]dto.UpdateColumnValueRequest, dto.CaseNormalizationResponse) {
	result := dto.CaseNormalizationResponse{
		TotalScanned: len(rows) * len(selectedColumns),
		TotalRows:    len(rows),
	}
	if len(rows) == 0 {
		return nil, result
	}

	updates := make([]dto.UpdateColumnValueRequest, 0)
	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalSkipped += len(selectedColumns)
			result.TotalRowsSkipped++
			continue
		}
		rowUpdates, skipped, rowUpdated := s.BuildCaseNormalizationUpdatesForRow(rowID, row, selectedColumns, caseFormat)
		if skipped > 0 {
			result.TotalSkipped += skipped
		}
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
			result.TotalUpdated += len(rowUpdates)
		}
		if rowUpdated {
			result.TotalRowsUpdated++
		} else {
			result.TotalRowsSkipped++
		}
	}
	return updates, result
}

func (s tableManagementService) BuildCaseNormalizationUpdatesForRow(rowID interface{}, row map[string]interface{}, selectedColumns []string, caseFormat string) ([]dto.UpdateColumnValueRequest, int, bool) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	skipped := 0
	rowUpdated := false
	for _, columnName := range selectedColumns {
		value, exists := row[columnName]
		if !exists || value == nil {
			skipped++
			continue
		}
		strValue, ok := value.(string)
		if !ok {
			skipped++
			continue
		}
		normalized := NormalizeValue(strValue, caseFormat)
		if normalized == strValue {
			skipped++
			continue
		}
		updates = append(updates, dto.UpdateColumnValueRequest{Id: rowID, Column: columnName, Value: normalized})
		rowUpdated = true
	}
	return updates, skipped, rowUpdated
}

func (s tableManagementService) BuildFindReplaceUpdates(rows []map[string]interface{}, selectedColumns []string, findValue, replaceValue, matchType string, ignoreRe *regexp.Regexp) ([]dto.UpdateColumnValueRequest, dto.FindReplaceResponse) {
	result := dto.FindReplaceResponse{
		TotalScanned: len(rows) * len(selectedColumns),
		TotalRows:    len(rows),
	}
	if len(rows) == 0 {
		return nil, result
	}

	updates := make([]dto.UpdateColumnValueRequest, 0)
	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalSkipped += len(selectedColumns)
			result.TotalRowsSkipped++
			continue
		}
		rowUpdates, skipped, rowUpdated, matchedCount, updatedCount := s.BuildFindReplaceUpdatesForRow(rowID, row, selectedColumns, findValue, replaceValue, matchType, ignoreRe)
		if skipped > 0 {
			result.TotalSkipped += skipped
		}
		if matchedCount > 0 {
			result.TotalMatched += matchedCount
		}
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
			result.TotalUpdated += updatedCount
		}
		if rowUpdated {
			result.TotalRowsUpdated++
		} else {
			result.TotalRowsSkipped++
		}
	}
	return updates, result
}

func (s tableManagementService) BuildFindReplaceUpdatesForRow(rowID interface{}, row map[string]interface{}, selectedColumns []string, findValue, replaceValue, matchType string, ignoreRe *regexp.Regexp) ([]dto.UpdateColumnValueRequest, int, bool, int, int) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	skipped := 0
	rowUpdated := false
	matched := 0
	updated := 0
	for _, columnName := range selectedColumns {
		value, exists := row[columnName]
		if !exists || value == nil {
			skipped++
			continue
		}
		strValue, ok := value.(string)
		if !ok {
			skipped++
			continue
		}
		isMatch, newVal := s.ComputeFindReplace(strValue, findValue, replaceValue, matchType, ignoreRe)
		if !isMatch {
			skipped++
			continue
		}
		matched++
		if newVal == strValue {
			continue
		}
		updates = append(updates, dto.UpdateColumnValueRequest{Id: rowID, Column: columnName, Value: newVal})
		updated++
		rowUpdated = true
	}
	return updates, skipped, rowUpdated, matched, updated
}

func (s tableManagementService) CaseNormalization(ctx context.Context, schemaName string, req dto.CaseNormalizationRequest) (dto.CaseNormalizationResponse, error) {

	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.CaseNormalizationResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.CaseNormalizationResponse{}, err
	}

	selectedColumns, err := s.GetSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.CaseNormalizationResponse{}, err
	}

	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rows, err := s.FetchTableRowsForTrim(ctx, tableName, selectColumns)
	if err != nil {
		return dto.CaseNormalizationResponse{}, err
	}

	updates, result := s.BuildCaseNormalizationUpdates(rows, selectedColumns, req.CaseFormat)

	if len(updates) > 0 {
		if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, model.Alias, updates); err != nil {
			return dto.CaseNormalizationResponse{}, err
		}
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int("total_scanned", result.TotalScanned).
		Int("total_updated", result.TotalUpdated).
		Int("total_skipped", result.TotalSkipped).
		Int("total_rows", result.TotalRows).
		Int("total_rows_updated", result.TotalRowsUpdated).
		Int("total_rows_skipped", result.TotalRowsSkipped).
		Str("case_format", req.CaseFormat).
		Msg("Case normalization action completed")

	return result, nil
}

func (s tableManagementService) FindReplace(ctx context.Context, schemaName string, req dto.FindReplaceRequest) (dto.FindReplaceResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.FindReplaceResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.FindReplaceResponse{}, err
	}

	selectedColumns, err := s.GetSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.FindReplaceResponse{}, err
	}

	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)

	batchSize := 1000
	limit := batchSize
	offset := 0

	totalResult := dto.FindReplaceResponse{}

	var ignoreRe *regexp.Regexp
	if req.MatchType == "ignore_case" {
		ignoreRe = regexp.MustCompile("(?i)" + regexp.QuoteMeta(req.FindValue))
	}

	for {
		params := dbModels.QueryParams{Select: selectColumns, Limit: &limit, Offset: &offset}
		rows, err := s.repo.TableService.GetTableData(tableName, params)
		if err != nil {
			return dto.FindReplaceResponse{}, app_errors.LogDatabaseError(err, "failed to fetch rows for find and replace")
		}

		updates, result := s.BuildFindReplaceUpdates(rows, selectedColumns, req.FindValue, req.ReplaceValue, req.MatchType, ignoreRe)

		if len(updates) > 0 {
			if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, model.Alias, updates); err != nil {
				return dto.FindReplaceResponse{}, err
			}
		}

		totalResult.TotalScanned += result.TotalScanned
		totalResult.TotalMatched += result.TotalMatched
		totalResult.TotalUpdated += result.TotalUpdated
		totalResult.TotalSkipped += result.TotalSkipped
		totalResult.TotalRows += result.TotalRows
		totalResult.TotalRowsUpdated += result.TotalRowsUpdated
		totalResult.TotalRowsSkipped += result.TotalRowsSkipped

		lg.Info().
			Str("model_id", req.ModelID).
			Int("batch_offset", offset).
			Int("batch_rows", len(rows)).
			Int("batch_updates", len(updates)).
			Msg("FindReplace batch processed")

		if len(rows) < limit {
			break
		}
		offset += limit
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int("total_scanned", totalResult.TotalScanned).
		Int("total_matched", totalResult.TotalMatched).
		Int("total_updated", totalResult.TotalUpdated).
		Int("total_skipped", totalResult.TotalSkipped).
		Int("total_rows", totalResult.TotalRows).
		Int("total_rows_updated", totalResult.TotalRowsUpdated).
		Int("total_rows_skipped", totalResult.TotalRowsSkipped).
		Str("match_type", req.MatchType).
		Str("find_value", req.FindValue).
		Msg("Find & Replace action completed")

	return totalResult, nil
}

func (s tableManagementService) BuildRemoveFormattingUpdates(rows []map[string]interface{}, formatting string, customPatterns []string, selectedColumns []string) ([]dto.UpdateColumnValueRequest, dto.RemoveFormattingResponse) {
	result := dto.RemoveFormattingResponse{ScannedRecords: len(rows) * len(selectedColumns)}
	if len(rows) == 0 {
		return nil, result
	}
	updates := make([]dto.UpdateColumnValueRequest, 0)
	for _, row := range rows {
		rowUpdates, rowStatus := s.BuildRemoveFormattingUpdatesForRow(row, formatting, customPatterns, selectedColumns)
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
		}
		switch {
		case rowStatus == removeFormattingRowUpdated:
			result.UpdatedRecords++
		case rowStatus == removeFormattingRowSkipped:
			result.SkippedRecords++
		}
	}
	return updates, result
}

type removeFormattingRowStatus int

func (s tableManagementService) BuildRemoveFormattingUpdatesForRow(row map[string]interface{}, formatting string, customPatterns []string, selectedColumns []string) ([]dto.UpdateColumnValueRequest, removeFormattingRowStatus) {
	rowID, hasRowID := row["id"]
	if !hasRowID {
		return nil, removeFormattingRowSkipped
	}

	logger.Get().
		Info().
		Interface("row_id", rowID).
		Str("formatting", formatting).
		Int("selected_columns", len(selectedColumns)).
		Msg("Processing row for remove formatting")

	updates := make([]dto.UpdateColumnValueRequest, 0)
	rowUpdated := false

	for _, columnName := range selectedColumns {
		if strings.EqualFold(strings.TrimSpace(columnName), "id") {
			continue
		}
		value, exists := LookupRowValue(row, columnName)
		if !exists || value == nil {
			continue
		}
		if upd, ok := s.ProcessRemoveFormattingCell(rowID, columnName, value, formatting, customPatterns); ok {
			updates = append(updates, *upd)
			rowUpdated = true
		}
	}

	if rowUpdated {
		return updates, removeFormattingRowUpdated
	}
	return updates, removeFormattingRowSkipped
}

func (s tableManagementService) ApplyRemoveFormattingUpdates(ctx context.Context, tableName string, updates []dto.UpdateColumnValueRequest) error {
	for _, update := range updates {
		rowID := update.Id
		updateData := map[string]interface{}{
			update.Column: update.Value,
		}
		logger.Get().
			Info().
			Interface("row_id", rowID).
			Str("table", tableName).
			Str("column", update.Column).
			Interface("value", update.Value).
			Msg("Applying remove formatting update")
		if _, err := s.repo.TableService.UpdateRecord(tableName, rowID, updateData); err != nil {
			logger.Get().
				Error().
				Interface("row_id", rowID).
				Str("table", tableName).
				Str("column", update.Column).
				Interface("value", update.Value).
				Err(err).
				Msg("Remove formatting update failed")
			return app_errors.LogDatabaseError(err, "failed to apply remove formatting updates")
		}
	}

	return nil
}

func (s tableManagementService) RemoveSpecialCharacters(ctx context.Context, schemaName string, req dto.RemoveSpecialCharactersRequest) (dto.RemoveSpecialCharactersResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveSpecialCharactersResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveSpecialCharactersResponse{}, err
	}

	selectedColumns, err := s.GetSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.RemoveSpecialCharactersResponse{}, err
	}

	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)

	limit := columnActionBatchSize
	offset := 0
	totalResult := dto.RemoveSpecialCharactersResponse{}

	for {
		params := dbModels.QueryParams{
			Select: selectColumns,
			Limit:  &limit,
			Offset: &offset,
		}
		rows, err := s.repo.TableService.GetTableData(tableName, params)
		if err != nil {
			return dto.RemoveSpecialCharactersResponse{}, app_errors.LogDatabaseError(err, "failed to fetch rows for remove special characters")
		}

		updates, result := s.BuildRemoveSpecialCharactersUpdates(rows, selectedColumns, req.SpecialCharactersType, req.CustomCharacter)

		if len(updates) > 0 {
			if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, model.Alias, updates); err != nil {
				return dto.RemoveSpecialCharactersResponse{}, err
			}
		}

		totalResult.TotalScanned += result.TotalScanned
		totalResult.TotalMatched += result.TotalMatched
		totalResult.TotalUpdated += result.TotalUpdated
		totalResult.TotalSkipped += result.TotalSkipped
		totalResult.TotalRows += result.TotalRows
		totalResult.TotalRowsUpdated += result.TotalRowsUpdated
		totalResult.TotalRowsSkipped += result.TotalRowsSkipped

		lg.Info().
			Str("model_id", req.ModelID).
			Int("batch_offset", offset).
			Int("batch_rows", len(rows)).
			Int("batch_updates", len(updates)).
			Msg("RemoveSpecialCharacters batch processed")

		if len(rows) < limit {
			break
		}
		offset += limit
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int("total_scanned", totalResult.TotalScanned).
		Int("total_matched", totalResult.TotalMatched).
		Int("total_updated", totalResult.TotalUpdated).
		Int("total_skipped", totalResult.TotalSkipped).
		Int("total_rows", totalResult.TotalRows).
		Int("total_rows_updated", totalResult.TotalRowsUpdated).
		Int("total_rows_skipped", totalResult.TotalRowsSkipped).
		Str("special_characters_type", req.SpecialCharactersType).
		Msg("Remove special characters action completed")

	return totalResult, nil
}

func (s tableManagementService) BuildRemoveSpecialCharactersUpdates(rows []map[string]interface{}, selectedColumns []string, removeType string, customChars []string) ([]dto.UpdateColumnValueRequest, dto.RemoveSpecialCharactersResponse) {
	result := dto.RemoveSpecialCharactersResponse{TotalScanned: len(rows) * len(selectedColumns), TotalRows: len(rows)}
	if len(rows) == 0 {
		return nil, result
	}
	updates := make([]dto.UpdateColumnValueRequest, 0)
	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalSkipped += len(selectedColumns)
			result.TotalRowsSkipped++
			continue
		}
		rowUpdates, skipped, rowUpdated, matchedCount, updatedCount := s.BuildRemoveSpecialCharactersUpdatesForRow(rowID, row, selectedColumns, removeType, customChars)
		if skipped > 0 {
			result.TotalSkipped += skipped
		}
		if matchedCount > 0 {
			result.TotalMatched += matchedCount
		}
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
			result.TotalUpdated += updatedCount
		}
		if rowUpdated {
			result.TotalRowsUpdated++
		} else {
			result.TotalRowsSkipped++
		}
	}
	return updates, result
}

func (s tableManagementService) BuildRemoveSpecialCharactersUpdatesForRow(rowID interface{}, row map[string]interface{}, selectedColumns []string, removeType string, customChars []string) ([]dto.UpdateColumnValueRequest, int, bool, int, int) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	skipped := 0
	rowUpdated := false
	matched := 0
	updated := 0
	for _, columnName := range selectedColumns {
		value, exists := row[columnName]
		if !exists || value == nil {
			skipped++
			continue
		}
		strValue, ok := value.(string)
		if !ok {
			skipped++
			continue
		}
		isMatch, newVal := ComputeRemoveSpecialCharacters(strValue, removeType, customChars)
		if !isMatch {
			skipped++
			continue
		}
		matched++
		if newVal == strValue {
			continue
		}
		updates = append(updates, dto.UpdateColumnValueRequest{Id: rowID, Column: columnName, Value: newVal})
		updated++
		rowUpdated = true
	}
	return updates, skipped, rowUpdated, matched, updated
}

func (s tableManagementService) RemoveDuplicates(ctx context.Context, schemaName string, req dto.RemoveDuplicatesRequest) (dto.RemoveDuplicatesResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveDuplicatesResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveDuplicatesResponse{}, err
	}

	selectedColumns, err := s.GetSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.RemoveDuplicatesResponse{}, err
	}

	if req.KeepRule == "keep_latest_updated" {
		supported, err := s.HasUpdateTimestampColumn(ctx, schemaName, model.Alias)
		if err != nil {
			return dto.RemoveDuplicatesResponse{}, err
		}
		if !supported {
			return dto.RemoveDuplicatesResponse{}, fmt.Errorf("%w: keep_latest_updated requires last_modified_time column on table %s", app_errors.InvalidPayload, model.Alias)
		}
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	affectedRows, totalDuplicateRows, err := s.ExecuteRemoveDuplicates(ctx, tableName, req.Duplicate, req.KeepRule, selectedColumns)
	if err != nil {
		return dto.RemoveDuplicatesResponse{}, err
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int64("total_rows_affected", affectedRows).
		Int64("total_duplicate_rows", totalDuplicateRows).
		Str("duplicate", req.Duplicate).
		Str("keep_rule", req.KeepRule).
		Msg("Remove duplicates action completed")

	return dto.RemoveDuplicatesResponse{TotalRowsAffected: affectedRows, TotalDuplicateRows: totalDuplicateRows}, nil
}

func (s tableManagementService) ColumnSplit(ctx context.Context, schemaName string, req dto.ColumnSplitRequest) (dto.ColumnSplitResponse, error) {
	lg := logger.Get()

	model, columnsData, selectedColumn, selectedIndex, selectedOrder, strategy, err := s.PrepareColumnSplit(ctx, schemaName, req)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	if strategy.kind == "fixed_length" {
		if err := s.ValidateFixedLengthBounds(ctx, schemaName, model.Alias, selectedColumn.ColumnName, strategy.value); err != nil {
			return dto.ColumnSplitResponse{}, err
		}
	}

	maxParts, err := s.GetMaxSplitParts(ctx, schemaName, model.Alias, selectedColumn.ColumnName, strategy)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	if err = EnsureSplitIsPossible(maxParts, selectedColumn.ColumnName); err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	columnCount, err := ResolveSplitColumnCount(maxParts, req.Limit)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	orderIndexes, err := s.ComputeSplitOrderIndexes(columnsData, selectedIndex, req.Where, columnCount)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	if req.Where == "next" {
		if err := s.ShiftColumnsForNext(ctx, schemaName, columnsData, selectedOrder, columnCount); err != nil {
			return dto.ColumnSplitResponse{}, err
		}
	}

	newColumnNames := make([]string, 0, columnCount)
	createdColumns := make([]tenant.Column, 0, columnCount)

	generatedNames, err := s.BuildSplitColumnNames(columnsData, selectedColumn.ColumnName, columnCount)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	createdColumns, newColumnNames, err = s.CreateSplitColumns(ctx, schemaName, model, selectedColumn, orderIndexes, generatedNames)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	finArgs := splitFinalizationArgs{
		SchemaName:     schemaName,
		ModelAlias:     model.Alias,
		SelectedColumn: selectedColumn,
		CreatedColumns: createdColumns,
		NewColumnNames: newColumnNames,
		Strategy:       strategy,
		ColumnCount:    columnCount,
		KeepOriginal:   req.KeepOriginal,
	}
	if err := s.FinalizeColumnSplit(ctx, finArgs); err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	lg.Info().Str("model_id", req.ModelID.String()).Str("column_id", req.ColumnID.String()).Str("column_name", selectedColumn.ColumnName).Str("split_type", strategy.kind).Int("created_columns", len(newColumnNames)).Bool("keep_original", req.KeepOriginal).Str("where", req.Where).Msg("Column split completed")
	return dto.ColumnSplitResponse{Message: "Column split completed successfully", CreatedColumns: newColumnNames}, nil
}

func (s tableManagementService) PrepareColumnSplit(ctx context.Context, schemaName string, req dto.ColumnSplitRequest) (tenant.Model, []dto.ColumnResponse, dto.ColumnResponse, int, float64, columnSplitStrategy, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID.String())
	if err != nil {
		return tenant.Model{}, nil, dto.ColumnResponse{}, -1, 0, columnSplitStrategy{}, err
	}
	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID.String())
	if err != nil {
		return tenant.Model{}, nil, dto.ColumnResponse{}, -1, 0, columnSplitStrategy{}, err
	}
	selectedColumn, selectedIndex, err := s.GetColumnByIDFromList(columnsData, req.ColumnID.String())
	if err != nil {
		return tenant.Model{}, nil, dto.ColumnResponse{}, -1, 0, columnSplitStrategy{}, err
	}
	selectedOrder := 0.0
	if selectedColumn.OrderIndex != nil {
		selectedOrder = *selectedColumn.OrderIndex
	}
	strategy, err := s.ParseColumnSplitStrategy(req.SplitBy)
	if err != nil {
		return tenant.Model{}, nil, dto.ColumnResponse{}, -1, 0, columnSplitStrategy{}, err
	}
	return model, columnsData, selectedColumn, selectedIndex, selectedOrder, strategy, nil
}

func (s tableManagementService) ValidateFixedLengthBounds(ctx context.Context, schemaName, tableName, columnName string, value int) error {
	query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE %s IS NOT NULL AND %s <> '' AND char_length(%s) < $1)`, fmt.Sprintf(SchemaTableFormat, schemaName, tableName), fmt.Sprintf(QuotedColumnFormat, columnName), fmt.Sprintf(QuotedColumnFormat, columnName), fmt.Sprintf(QuotedColumnFormat, columnName))
	var exceeds bool
	rows, queryErr := s.repo.DB.QueryContext(ctx, query, value)
	if queryErr != nil {
		return app_errors.LogDatabaseError(queryErr, "failed to validate fixed length bounds")
	}
	defer rows.Close()
	if rows.Next() {
		if scanErr := rows.Scan(&exceeds); scanErr != nil {
			return app_errors.LogDatabaseError(scanErr, "failed to scan fixed length bounds")
		}
	}
	if exceeds {
		return fmt.Errorf("%w: fixed_length value cannot exceed string length", app_errors.InvalidPayload)
	}
	return nil
}

func (s tableManagementService) GetMaxSplitParts(ctx context.Context, schemaName, tableName, columnName string, strategy columnSplitStrategy) (int, error) {
	arrayExpr, params := s.GetSplitSQLArrayExpr(columnName, strategy)
	maxPartsQuery := fmt.Sprintf(`SELECT COALESCE(MAX(array_length(%s, 1)), 0) FROM %s`, arrayExpr, fmt.Sprintf(SchemaTableFormat, schemaName, tableName))
	var maxParts int
	rows, queryErr := s.repo.DB.QueryContext(ctx, maxPartsQuery, params...)
	if queryErr != nil {
		return 0, app_errors.LogDatabaseError(queryErr, "failed to calculate maximum split parts")
	}
	defer rows.Close()
	if rows.Next() {
		if scanErr := rows.Scan(&maxParts); scanErr != nil {
			return 0, app_errors.LogDatabaseError(scanErr, "failed to scan maximum split parts")
		}
	}
	if maxParts <= 0 {
		return 0, fmt.Errorf("%w: no values available to split in column %s", app_errors.InvalidPayload, columnName)
	}
	return maxParts, nil
}

func (s tableManagementService) ShiftColumnsForNext(ctx context.Context, schemaName string, columnsData []dto.ColumnResponse, selectedOrder float64, columnCount int) error {
	affected := make([]dto.ColumnResponse, 0)
	for _, col := range columnsData {
		if col.OrderIndex != nil && *col.OrderIndex > selectedOrder {
			affected = append(affected, col)
		}
	}
	sort.SliceStable(affected, func(i, j int) bool { return *affected[i].OrderIndex > *affected[j].OrderIndex })
	for _, col := range affected {
		newOrderIndex := *col.OrderIndex + float64(columnCount)
		orderIndexUpdate := dto.ColumnUpdate{OrderIndex: &newOrderIndex}
		_, err := s.UpdateColumn(ctx, schemaName, col.ID.String(), orderIndexUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s tableManagementService) CreateSplitColumns(ctx context.Context, schemaName string, model tenant.Model, selectedColumn dto.ColumnResponse, orderIndexes []float64, generatedNames []string) ([]tenant.Column, []string, error) {
	createdColumns := make([]tenant.Column, 0, len(orderIndexes))
	newColumnNames := make([]string, 0, len(orderIndexes))
	now := time.Now().UTC()
	for idx, orderIndex := range orderIndexes {
		title := s.BuildSplitColumnTitle(selectedColumn.Title, selectedColumn.ColumnName, idx+1)
		colInsert := dto.ColumnInsertion{
			ID: uuid.New(), ModelID: model.ID, BaseID: model.BaseID, Title: title,
			ColumnName: generatedNames[idx], Description: &selectedColumn.Description, Meta: map[string]interface{}{},
			UIDT: "longText", DT: helpers.StringPtr("TEXT"), Virtual: false, System: false, Deleted: false,
			OrderIndex: &orderIndex, CreatedBy: "", UpdatedBy: "", CreatedAt: now, UpdatedAt: now,
		}
		createdCol, createErr := s.columnsService.Create(ctx, colInsert, schemaName)
		if createErr != nil {
			for _, c := range createdColumns {
				var dtoCol dto.ColumnResponse
				_ = helpers.StructToStruct(c, &dtoCol)
				_ = s.DeleteColumnAndCleanUp(ctx, schemaName, c.ID.String(), dtoCol)
			}
			return nil, nil, createErr
		}
		if addErr := s.AddColumnInTableDb(schemaName, model.Alias, createdCol); addErr != nil {
			var dtoCol dto.ColumnResponse
			_ = helpers.StructToStruct(createdCol, &dtoCol)
			_ = s.DeleteColumnAndCleanUp(ctx, schemaName, createdCol.ID.String(), dtoCol)
			for _, c := range createdColumns {
				var dc dto.ColumnResponse
				_ = helpers.StructToStruct(c, &dc)
				_ = s.DeleteColumnAndCleanUp(ctx, schemaName, c.ID.String(), dc)
			}
			return nil, nil, addErr
		}
		createdColumns = append(createdColumns, createdCol)
		newColumnNames = append(newColumnNames, createdCol.ColumnName)
	}
	return createdColumns, newColumnNames, nil
}

type splitFinalizationArgs struct {
	SchemaName     string
	ModelAlias     string
	SelectedColumn dto.ColumnResponse
	CreatedColumns []tenant.Column
	NewColumnNames []string
	Strategy       columnSplitStrategy
	ColumnCount    int
	KeepOriginal   bool
}

func (s tableManagementService) FinalizeColumnSplit(ctx context.Context, args splitFinalizationArgs) error {
	if err := s.PerformBulkSplitUpdate(ctx, args.SchemaName, args.ModelAlias, args.SelectedColumn.ColumnName, args.NewColumnNames, args.Strategy, args.ColumnCount); err != nil {
		for _, c := range args.CreatedColumns {
			var dtoCol dto.ColumnResponse
			_ = helpers.StructToStruct(c, &dtoCol)
			_ = s.DeleteColumnAndCleanUp(ctx, args.SchemaName, c.ID.String(), dtoCol)
		}
		return err
	}

	if !args.KeepOriginal {
		tx, startErr := s.repo.DB.Begin()
		if startErr != nil {
			return app_errors.LogDatabaseError(startErr, "failed to start transaction for column drop")
		}
		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback()
			}
		}()
		if err := s.DeleteSplitOriginalColumn(tx, args.SchemaName, args.ModelAlias, args.SelectedColumn); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return app_errors.LogDatabaseError(err, "failed to commit column split transaction")
		}
	}
	return nil
}

func (s tableManagementService) ExecuteRemoveDuplicates(txCtx context.Context, tableName string, Duplicate string, keepRule string, selectedColumns []string) (int64, int64, error) {
	matchCase := s.DetermineMatchCase(Duplicate)
	partitionBy, notAllSelectedColsEmpty := s.BuildDuplicateKeyExpressions(selectedColumns, matchCase)
	orderBy := s.BuildDuplicateKeepOrderBy(keepRule)
	totalDuplicateRows, err := s.CountDuplicateRows(txCtx, tableName, partitionBy, notAllSelectedColsEmpty)
	if err != nil {
		return 0, 0, err
	}
	switch Duplicate {
	case "remove_row":
		q := s.BuildDeleteDuplicatesQuery(tableName, partitionBy, orderBy, notAllSelectedColsEmpty)
		affected, err := s.ExecQueryAndRowsAffected(txCtx, q, "failed to remove duplicate rows")
		if err != nil {
			return 0, 0, err
		}
		return affected, totalDuplicateRows, nil
	case "remove_duplicates", "remove_duplicates_matchCase":
		setClauses := make([]string, 0, len(selectedColumns))
		for _, columnName := range selectedColumns {
			setClauses = append(setClauses, fmt.Sprintf("%s = NULL", fmt.Sprintf(QuotedColumnFormat, columnName)))
		}
		q := s.BuildUpdateDuplicatesQuery(tableName, partitionBy, orderBy, notAllSelectedColsEmpty, strings.Join(setClauses, ", "))
		affected, err := s.ExecQueryAndRowsAffected(txCtx, q, "failed to clear duplicate values")
		if err != nil {
			return 0, 0, err
		}
		return affected, totalDuplicateRows, nil
	default:
		return 0, 0, fmt.Errorf("%w: unsupported duplicate handling mode %s", app_errors.InvalidPayload, Duplicate)
	}
}

func (s tableManagementService) DetermineMatchCase(mode string) bool {
	switch mode {
	case "remove_duplicates_matchCase":
		return true
	case "remove_row", "remove_duplicates":
		return false
	default:
		return true
	}
}

func (s tableManagementService) CountDuplicateRows(ctx context.Context, tableName, partitionBy, condition string) (int64, error) {
	query := fmt.Sprintf(`SELECT COALESCE(SUM(cnt), 0) FROM (SELECT COUNT(*) AS cnt FROM %s WHERE %s GROUP BY %s HAVING COUNT(*) > 1) t;`, tableName, condition, partitionBy)
	var total int64
	rows, err := s.repo.DB.QueryContext(ctx, query)
	if err != nil {
		return 0, app_errors.LogDatabaseError(err, "failed to count duplicate rows")
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, app_errors.LogDatabaseError(err, "failed to read duplicate rows count")
		}
	}
	return total, nil
}

func (s tableManagementService) ExecQueryAndRowsAffected(ctx context.Context, query, errMsg string) (int64, error) {
	result, err := s.repo.DB.ExecContext(ctx, query)
	if err != nil {
		return 0, app_errors.LogDatabaseError(err, errMsg)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, app_errors.LogDatabaseError(err, "failed to get rows affected")
	}
	return affected, nil
}

func (s tableManagementService) BuildDeleteDuplicatesQuery(tableName, partitionBy, orderBy, condition string) string {
	return fmt.Sprintf(`WITH duplicates AS (
			SELECT id,
				   ROW_NUMBER() OVER (PARTITION BY %s ORDER BY %s) AS row_number
			FROM %s
			WHERE %s
		)
		DELETE FROM %s
		WHERE id IN (
			SELECT id FROM duplicates WHERE row_number > 1
		);`, partitionBy, orderBy, tableName, condition, tableName)
}

func (s tableManagementService) BuildUpdateDuplicatesQuery(tableName, partitionBy, orderBy, condition, setClause string) string {
	return fmt.Sprintf(`WITH duplicates AS (
			SELECT id,
				   ROW_NUMBER() OVER (PARTITION BY %s ORDER BY %s) AS row_number
			FROM %s
			WHERE %s
		)
		UPDATE %s target
		SET %s
		FROM duplicates d
		WHERE target.id = d.id
		  AND d.row_number > 1;`, partitionBy, orderBy, tableName, condition, tableName, setClause)
}

func (s tableManagementService) BuildDuplicateKeyExpressions(selectedColumns []string, matchCase bool) (string, string) {
	expressions := make([]string, 0, len(selectedColumns))
	nullChecks := make([]string, 0, len(selectedColumns))
	for _, columnName := range selectedColumns {
		baseExpr := fmt.Sprintf("NULLIF(TRIM(CAST(%s AS TEXT)), '')", fmt.Sprintf(QuotedColumnFormat, columnName))
		expr := baseExpr
		if !matchCase {
			expr = fmt.Sprintf("LOWER(%s)", baseExpr)
		}
		expressions = append(expressions, expr)
		nullChecks = append(nullChecks, fmt.Sprintf("%s IS NULL", expr))
	}
	return strings.Join(expressions, ", "), fmt.Sprintf("NOT (%s)", strings.Join(nullChecks, " AND "))
}

func (s tableManagementService) BuildDuplicateKeepOrderBy(keepRule string) string {
	switch keepRule {
	case "keep_last":
		return fmt.Sprintf("%s DESC", fmt.Sprintf(QuotedColumnFormat, "id"))
	case "keep_latest_updated":
		return fmt.Sprintf("%s DESC, %s DESC", fmt.Sprintf(QuotedColumnFormat, "last_modified_time"), fmt.Sprintf(QuotedColumnFormat, "id"))
	default:
		return fmt.Sprintf("%s ASC", fmt.Sprintf(QuotedColumnFormat, "id"))
	}
}

func (s tableManagementService) HasUpdateTimestampColumn(ctx context.Context, schemaName, tableName string) (bool, error) {
	filters := []dbModels.QueryFilter{{Column: "table_schema", Operator: "=", Value: schemaName}, {Column: "table_name", Operator: "=", Value: tableName}}
	params := dbModels.QueryParams{Select: []string{"column_name"}, Filters: filters}
	rows, err := s.repo.TableService.GetTableData("information_schema.columns", params)
	if err != nil {
		return false, app_errors.LogDatabaseError(err, "failed to inspect table columns")
	}
	for _, row := range rows {
		if colName, ok := row["column_name"].(string); ok && colName == "last_modified_time" {
			return true, nil
		}
	}
	return false, nil
}

func (s tableManagementService) ParseColumnSplitStrategy(splitBy dto.SplitByRequest) (columnSplitStrategy, error) {
	kind := strings.TrimSpace(strings.ToLower(splitBy.Type))
	switch kind {
	case "separator":
		return ParseSeparator(splitBy)
	case "fixedlength":
		return ParseFixedLength(splitBy)
	case "pattern":
		return ParsePattern(splitBy)
	default:
		return columnSplitStrategy{}, fmt.Errorf("%w: unsupported split type %s", app_errors.InvalidPayload, splitBy.Type)
	}
}

func ParseSeparator(splitBy dto.SplitByRequest) (columnSplitStrategy, error) {
	separator, _ := splitBy.Config["separator"].(string)
	if separator == "" {
		return columnSplitStrategy{}, fmt.Errorf("%w: separator cannot be empty", app_errors.InvalidPayload)
	}
	return columnSplitStrategy{kind: "separator", separator: separator}, nil
}

func ParseFixedLength(splitBy dto.SplitByRequest) (columnSplitStrategy, error) {
	action, _ := splitBy.Config["action"].(string)
	action = strings.TrimSpace(strings.ToLower(action))
	if action != "after" && action != "before" {
		return columnSplitStrategy{}, fmt.Errorf("%w: fixed_length action must be after or before", app_errors.InvalidPayload)
	}
	value, ok := splitBy.Config["value"]
	if !ok {
		return columnSplitStrategy{}, fmt.Errorf("%w: fixed_length value is required", app_errors.InvalidPayload)
	}
	valueInt, err := ParsePositiveSplitInt(value)
	if err != nil {
		return columnSplitStrategy{}, err
	}
	return columnSplitStrategy{kind: "fixed_length", action: action, value: valueInt}, nil
}

func ParsePattern(splitBy dto.SplitByRequest) (columnSplitStrategy, error) {
	pattern, _ := splitBy.Config["pattern"].(string)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return columnSplitStrategy{}, fmt.Errorf("%w: pattern cannot be empty", app_errors.InvalidPayload)
	}
	allowed := map[string]struct{}{"\\d+": {}, "[A-Z]+": {}, "[a-z]+": {}, "[A-Za-z]+": {}, "\\s+": {}, "[^a-zA-Z0-9]": {}, "@(.+)": {}, "\\.": {}}
	if _, ok := allowed[pattern]; !ok {
		return columnSplitStrategy{}, fmt.Errorf("%w: unsupported regex pattern", app_errors.InvalidPayload)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return columnSplitStrategy{}, fmt.Errorf("%w: invalid regex pattern: %v", app_errors.InvalidPayload, err)
	}
	return columnSplitStrategy{kind: "pattern", pattern: pattern, regex: re}, nil
}

func InvalidPayload(msg string) error {
	return fmt.Errorf("%w: %s", app_errors.InvalidPayload, msg)
}

func ParsePositiveSplitInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		if v <= 0 {
			return 0, InvalidPayload(errFixedLengthPositive)
		}
		return v, nil
	case int32:
		if v <= 0 {
			return 0, InvalidPayload(errFixedLengthPositive)
		}
		return int(v), nil
	case int64:
		if v <= 0 {
			return 0, InvalidPayload(errFixedLengthPositive)
		}
		return int(v), nil
	case float32:
		if v <= 0 {
			return 0, InvalidPayload(errFixedLengthPositive)
		}
		return int(v), nil
	case float64:
		if v <= 0 {
			return 0, InvalidPayload(errFixedLengthPositive)
		}
		return int(v), nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || parsed <= 0 {
			return 0, InvalidPayload(errFixedLengthPositive)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("%w: %s", app_errors.InvalidPayload, errFixedLengthInvalid)
	}
}
func (s tableManagementService) GetColumnByIDFromList(columns []dto.ColumnResponse, id string) (dto.ColumnResponse, int, error) {
	for idx, col := range columns {
		if col.ID.String() == id {
			return col, idx, nil
		}
	}
	return dto.ColumnResponse{}, -1, app_errors.ColumnNotFound
}

func (s tableManagementService) BuildSplitColumnNames(columns []dto.ColumnResponse, baseName string, count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("%w: split count must be greater than zero", app_errors.InvalidPayload)
	}
	existing := make(map[string]struct{}, len(columns))
	for _, col := range columns {
		existing[col.ColumnName] = struct{}{}
	}
	names := make([]string, 0, count)
	nextSuffix := 1
	for len(names) < count {
		name := fmt.Sprintf("%s_%d", baseName, nextSuffix)
		for {
			if _, ok := existing[name]; !ok {
				break
			}
			nextSuffix++
			name = fmt.Sprintf("%s_%d", baseName, nextSuffix)
		}
		existing[name] = struct{}{}
		names = append(names, name)
		nextSuffix++
	}
	return names, nil
}

func (s tableManagementService) BuildSplitColumnTitle(baseTitle, baseName string, index int) string {
	seed := strings.TrimSpace(baseTitle)
	if seed == "" {
		seed = baseName
	}
	return fmt.Sprintf("%s_%d", seed, index)
}

func (s tableManagementService) ComputeSplitOrderIndexes(columns []dto.ColumnResponse, selectedIndex int, where string, count int) ([]float64, error) {
	if count <= 0 {
		return nil, fmt.Errorf("%w: split count must be greater than zero", app_errors.InvalidPayload)
	}
	selectedOrder := 0.0
	if columns[selectedIndex].OrderIndex != nil {
		selectedOrder = *columns[selectedIndex].OrderIndex
	}
	orderIndexes := make([]float64, 0, count)
	switch where {
	case "next":
		start := selectedOrder + 1
		for i := 0; i < count; i++ {
			orderIndexes = append(orderIndexes, start+float64(i))
		}
	case "end":
		maxOrder := selectedOrder
		for _, col := range columns {
			if col.OrderIndex != nil && *col.OrderIndex > maxOrder {
				maxOrder = *col.OrderIndex
			}
		}
		start := maxOrder + 1
		for i := 0; i < count; i++ {
			orderIndexes = append(orderIndexes, start+float64(i))
		}
	default:
		return nil, fmt.Errorf("%w: unsupported column placement %s", app_errors.InvalidPayload, where)
	}
	return orderIndexes, nil
}

func EnsureSplitIsPossible(maxParts int, columnName string) error {
	if maxParts <= 1 {
		return app_errors.SplitNotPossible
	}
	return nil
}

func ResolveSplitColumnCount(maxParts int, limit *int) (int, error) {
	columnCount := maxParts
	if limit != nil {
		if *limit < maxParts {
			columnCount = *limit
		}
	}
	if columnCount <= 1 {
		return 0, app_errors.SplitNotPossible
	}
	return columnCount, nil
}
func SplitJoinSeparator(strategy columnSplitStrategy) string {
	if strategy.kind == "separator" {
		return strategy.separator
	}
	return ""
}
func ApplySplitColumnLimit(parts []string, columnCount int, joinSeparator string) []string {
	if columnCount <= 0 || len(parts) <= columnCount {
		return parts
	}
	result := make([]string, 0, columnCount)
	result = append(result, parts[:columnCount-1]...)
	result = append(result, strings.Join(parts[columnCount-1:], joinSeparator))
	return result
}

// Public wrappers for additional helpers safe to call without DB dependencies
func FindLastSelectedOrderIndexPublic(columnsData []dto.ColumnResponse, lastSel string) (float64, bool) {
	return FindLastSelectedOrderIndex(columnsData, lastSel)
}

func ApplyBulkUpdatesPublic(columnsService interfaces.ColumnService, ctx context.Context, schemaName, modelAlias string, updates []dto.UpdateColumnValueRequest) error {
	if len(updates) == 0 {
		return nil
	}
	var s tableManagementService
	s.columnsService = columnsService
	return s.ApplyBulkUpdates(ctx, schemaName, modelAlias, updates)
}

func DeleteOriginalColumnsIfNeededPublic(svc interfaces.TableManagementService, ctx context.Context, schemaName string, req dto.MergeColumnsRequest, columnsData []dto.ColumnResponse) error {
	if len(req.Columns) == 0 {
		return nil
	}
	ts, ok := svc.(*tableManagementService)
	if !ok {
		return fmt.Errorf("DeleteOriginalColumnsIfNeededPublic: unsupported service implementation")
	}
	return ts.DeleteOriginalColumnsIfNeeded(ctx, schemaName, req, columnsData)
}

func (s tableManagementService) DeleteSplitOriginalColumn(tx *sql.Tx, schemaName, tableName string, column dto.ColumnResponse) error {
	alterQuery := fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, formatQualifiedTable(schemaName, tableName), quoteIdentifier(column.ColumnName))
	if _, err := tx.ExecContext(context.Background(), alterQuery); err != nil {
		return app_errors.LogDatabaseError(err, "failed to drop original split column")
	}
	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, formatQualifiedTable(schemaName, "columns"))
	if _, err := tx.ExecContext(context.Background(), deleteQuery, column.ID.String()); err != nil {
		return app_errors.LogDatabaseError(err, "failed to remove original split column metadata")
	}
	return nil
}

func (s tableManagementService) GetSplitSQLArrayExpr(columnName string, strategy columnSplitStrategy) (string, []interface{}) {
	quotedCol := fmt.Sprintf(QuotedColumnFormat, columnName)
	switch strategy.kind {
	case "separator":
		pattern := fmt.Sprintf("(?:%s)+", regexp.QuoteMeta(strategy.separator))
		return fmt.Sprintf("ARRAY(SELECT x FROM unnest(regexp_split_to_array(COALESCE(%s, ''), $1)) x WHERE TRIM(x) <> '')", quotedCol), []interface{}{pattern}
	case "fixed_length":
		var partsExpr string
		if strategy.action == "after" {
			partsExpr = fmt.Sprintf("ARRAY[substring(COALESCE(%s, '') from 1 for %d), substring(COALESCE(%s, '') from %d)]", quotedCol, strategy.value, quotedCol, strategy.value+1)
		} else {
			partsExpr = fmt.Sprintf(`CASE 
					WHEN char_length(COALESCE(%s, '')) < %d THEN ARRAY[]::text[]
					ELSE ARRAY[
						substring(COALESCE(%s, '') from 1 for char_length(COALESCE(%s, '')) - %d), 
						substring(COALESCE(%s, '') from char_length(COALESCE(%s, '')) - %d + 1)
					]
				 END`, quotedCol, strategy.value, quotedCol, quotedCol, strategy.value, quotedCol, quotedCol, strategy.value)
		}
		return fmt.Sprintf("ARRAY(SELECT x FROM unnest(%s) x WHERE TRIM(x) <> '')", partsExpr), nil
	case "pattern":
		return fmt.Sprintf("ARRAY(SELECT x FROM unnest(regexp_split_to_array(COALESCE(%s, ''), $1)) x WHERE TRIM(x) <> '')", quotedCol), []interface{}{strategy.pattern}
	default:
		return "ARRAY[]::text[]", nil
	}
}

func (s tableManagementService) PerformBulkSplitUpdate(ctx context.Context, schemaName string, tableName string, columnName string, newColumnNames []string, strategy columnSplitStrategy, columnCount int) error {
	fullTableName := fmt.Sprintf(SchemaTableFormat, schemaName, tableName)
	params := dbModels.QueryParams{Select: []string{"id", columnName}}
	rows, err := s.repo.TableService.GetTableData(fullTableName, params)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to fetch rows for column split")
	}
	batchSize := 2000
	updates := make([]dto.UpdateColumnValueRequest, 0, batchSize*columnCount)
	for idx, row := range rows {
		rowUpdates := BuildSplitUpdatesForRow(row, columnName, newColumnNames, strategy, columnCount)
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
		}
		if (idx+1)%batchSize == 0 {
			if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, tableName, updates); err != nil {
				return err
			}
			updates = updates[:0]
		}
	}
	if len(updates) > 0 {
		if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, tableName, updates); err != nil {
			return err
		}
	}
	return nil
}

func SplitStringInGo(val string, strategy columnSplitStrategy) []string {
	switch strategy.kind {
	case "separator":
		return SplitBySeparator(val, strategy.separator)
	case "fixed_length":
		return SplitByFixedLength(val, strategy.action, strategy.value)
	case "pattern":
		return SplitByPattern(val, strategy.regex)
	default:
		return nil
	}
}
func SplitBySeparator(val, separator string) []string {
	pattern := fmt.Sprintf("(?:%s)+", regexp.QuoteMeta(separator))
	re := regexp.MustCompile(pattern)
	rawParts := re.Split(val, -1)
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
func SplitByFixedLength(val, action string, value int) []string {
	runes := []rune(val)
	if len(runes) < value {
		if len(runes) > 0 {
			return []string{string(runes)}
		}
		return nil
	}
	if action == "after" {
		return FilterEmpty([]string{string(runes[:value]), string(runes[value:])})
	}
	splitIdx := len(runes) - value
	return FilterEmpty([]string{string(runes[:splitIdx]), string(runes[splitIdx:])})
}
func SplitByPattern(val string, re *regexp.Regexp) []string {
	if re == nil {
		return nil
	}
	rawParts := re.Split(val, -1)
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
func FilterEmpty(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

func BuildSplitUpdatesForRow(row map[string]interface{}, columnName string, newColumnNames []string, strategy columnSplitStrategy, columnCount int) []dto.UpdateColumnValueRequest {
	rowID, hasRowID := row["id"]
	if !hasRowID {
		return nil
	}
	var valStr string
	if valRaw, ok := row[columnName]; ok && valRaw != nil {
		if val, ok := valRaw.(string); ok {
			valStr = val
		}
	}
	rawParts := SplitStringInGo(valStr, strategy)
	parts := ApplySplitColumnLimit(rawParts, columnCount, SplitJoinSeparator(strategy))
	updates := make([]dto.UpdateColumnValueRequest, 0, columnCount)
	for colIdx := 0; colIdx < columnCount; colIdx++ {
		var valToSet interface{}
		if colIdx < len(parts) {
			valToSet = parts[colIdx]
		} else {
			valToSet = nil
		}
		updates = append(updates, dto.UpdateColumnValueRequest{Id: rowID, Column: newColumnNames[colIdx], Value: valToSet})
	}
	return updates
}

func (s tableManagementService) MergeColumns(ctx context.Context, schemaName string, req dto.MergeColumnsRequest) (dto.MergeColumnsResponse, error) {
	lg := logger.Get()
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}
	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}
	selectedColumns, err := s.GetSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}
	sep := DetermineMergeSeparator(req.MergeFormat, req.CustomSeparator)
	baseTitle := strings.TrimSpace(req.NewColumnTitle)
	if baseTitle == "" {
		baseTitle = CombineColumnTitles(selectedColumns, columnsData)
	}
	colTitle := UniqueTitleFromBase(baseTitle, columnsData)
	baseName := s.slugify(colTitle)
	uniqueName := UniqueNameFromBase(baseName, columnsData)
	desiredOrderIndex, err := s.DetermineDesiredOrderIndex(ctx, schemaName, req, columnsData)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}
	newCol, err := s.CreateNewColumnForMerge(ctx, schemaName, model, req, uniqueName, colTitle, desiredOrderIndex)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}
	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)
	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rows, err := s.FetchTableRowsForTrim(ctx, tableName, selectColumns)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}
	updates, result := s.BuildMergeUpdates(rows, selectedColumns, sep, newCol.ColumnName)
	if err := s.ApplyBulkUpdates(ctx, schemaName, model.Alias, updates); err != nil {
		return dto.MergeColumnsResponse{}, err
	}
	if !req.KeepOriginalColumn {
		if err := s.DeleteOriginalColumnsIfNeeded(ctx, schemaName, req, columnsData); err != nil {
			return dto.MergeColumnsResponse{}, err
		}
	}
	lg.Info().Str("model_id", req.ModelID).Int("columns_selected", len(selectedColumns)).Int("total_scanned", result.TotalScanned).Int("total_updated", result.TotalUpdated).Int("total_skipped", result.TotalSkipped).Int("total_rows", result.TotalRows).Int("total_rows_updated", result.TotalRowsUpdated).Int("total_rows_skipped", result.TotalRowsSkipped).Str("generated_column", newCol.ColumnName).Msg("Merge columns action completed")
	result.GeneratedColumn = newCol.ColumnName
	return result, nil
}

func (s tableManagementService) ExtractSubstring(ctx context.Context, schemaName string, req dto.ExtractSubstringRequest) (dto.ExtractSubstringResponse, error) {
	lg := logger.Get()
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}
	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}
	selectedCol, found := FindColumnByID(columnsData, req.ColumnId)
	if !found {
		return dto.ExtractSubstringResponse{}, app_errors.ColumnNotFound
	}
	effectiveType, err := ValidateExtractSubstringRequest(req)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}
	sourceColumnName := selectedCol.ColumnName
	// resolve or create the generated column (preferring type-specific extracted_<type>)
	desiredBaseName := fmt.Sprintf("Extracted_%s", effectiveType)
	generatedColumnName, err := s.ResolveOrCreateExtractedColumn(ctx, schemaName, model, columnsData, desiredBaseName, req)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}
	selectColumns := []string{"id", sourceColumnName}
	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rows, err := s.FetchTableRowsForTrim(ctx, tableName, selectColumns)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}
	result := dto.ExtractSubstringResponse{Column: sourceColumnName, GeneratedColumn: generatedColumnName, ExtractionType: effectiveType, ScannedRecords: len(rows)}
	updates, updatedCount, skippedCount := s.BuildExtractSubstringUpdates(rows, sourceColumnName, generatedColumnName, effectiveType, req)
	result.UpdatedRecords = updatedCount
	result.SkippedRecords = skippedCount
	if len(updates) > 0 {
		if err := s.ApplyBulkUpdates(ctx, schemaName, model.Alias, updates); err != nil {
			return dto.ExtractSubstringResponse{}, err
		}
	}
	if !req.KeepOriginalColumn {
		if err := s.DeleteColumnAndCleanUp(ctx, schemaName, selectedCol.ID.String(), selectedCol); err != nil {
			return dto.ExtractSubstringResponse{}, err
		}
	}
	lg.Info().Str("model_id", req.ModelID).Str("source_column", sourceColumnName).Str("generated_column", generatedColumnName).Int("scanned_records", result.ScannedRecords).Int("updated_records", result.UpdatedRecords).Int("skipped_records", result.SkippedRecords).Msg("Extract substring action completed")
	return result, nil
}

// resolveOrCreateExtractedColumn ensures a column with the desired name exists and returns its column name.
func (s tableManagementService) ResolveOrCreateExtractedColumn(ctx context.Context, schemaName string, model tenant.Model, columnsData []dto.ColumnResponse, desiredName string, req dto.ExtractSubstringRequest) (string, error) {
	// always create a new column based on desiredName; UniqueNameFromBase will append numbers if needed
	baseName := desiredName
	uniqueName := UniqueNameFromBase(baseName, columnsData)
	// derive friendly title from extraction type, e.g. "Extracted Email"
	extractedType := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(desiredName)), "extracted_")
	titleMap := map[string]string{
		"email":    "Extracted Email",
		"url":      "Extracted URL",
		"domain":   "Extracted Domain",
		"keywords": "Extracted Keywords",
		"mentions": "Extracted Mentions",
		"tags":     "Extracted Tags",
		"emoji":    "Extracted Emoji",
		"phone":    "Extracted Phone",
		"prefix":   "Extracted Prefix",
	}
	baseTitle, ok := titleMap[extractedType]
	if !ok || strings.TrimSpace(baseTitle) == "" {
		baseTitle = "Extracted value"
	}
	colTitle := UniqueTitleFromBase(baseTitle, columnsData)
	mergeReq := dto.MergeColumnsRequest{ModelID: req.ModelID, Columns: []string{req.ColumnId}, AddAtEnd: req.AddAtEnd}
	desiredOrderIndex, err := s.DetermineDesiredOrderIndex(ctx, schemaName, mergeReq, columnsData)
	if err != nil {
		return "", err
	}
	newCol, err := s.CreateNewColumnForMerge(ctx, schemaName, model, mergeReq, uniqueName, colTitle, desiredOrderIndex)
	if err != nil {
		return "", err
	}
	return newCol.ColumnName, nil
}

func FindColumnByID(columnsData []dto.ColumnResponse, columnID string) (dto.ColumnResponse, bool) {
	targetID := strings.TrimSpace(columnID)
	for _, c := range columnsData {
		if c.ID.String() == targetID {
			return c, true
		}
	}
	return dto.ColumnResponse{}, false
}

func ValidateExtractSubstringRequest(req dto.ExtractSubstringRequest) (string, error) {
	method := strings.ToLower(strings.TrimSpace(req.ExtractionMethod))
	switch method {
	case "extraction_type":
		if strings.TrimSpace(req.ExtractionType) == "" {
			return "", app_errors.InvalidPayload
		}
		allowed := map[string]struct{}{"email": {}, "keywords": {}, "mentions": {}, "tags": {}, "url": {}, "domain": {}, "emoji": {}, "phone": {}, "prefix": {}}
		effectiveType := strings.ToLower(strings.TrimSpace(req.ExtractionType))
		if _, ok := allowed[effectiveType]; !ok {
			return "", app_errors.InvalidPayload
		}
		return effectiveType, nil
	case "between_characters":
		if strings.TrimSpace(req.StartAfter) == "" || strings.TrimSpace(req.EndBefore) == "" {
			return "", app_errors.InvalidPayload
		}
		return "between_characters", nil
	default:
		return "", app_errors.InvalidPayload
	}
}

func ExtractSubstringByType(strVal, extractionType, startAfter, endBefore string) (string, bool) {
	switch extractionType {
	case "between_characters":
		return ExtractBetweenCharactersFromText(strVal, startAfter, endBefore)
	case "email":
		return ExtractFirstEmail(strVal)
	case "url":
		return ExtractURLsFromText(strVal)
	case "domain":
		return ExtractDomainFromText(strVal)
	case "tags":
		return ExtractHashtagsFromText(strVal)
	case "mentions":
		return ExtractMentionsFromText(strVal)
	case "keywords":
		return ExtractKeywordsFromText(strVal)
	case "emoji":
		return ExtractEmojiFromText(strVal)
	case "phone":
		return ExtractPhoneNumberFromText(strVal)
	case "prefix":
		return ExtractEmailPrefixFromText(strVal)
	default:
		return "", false
	}
}

// BuildExtractSubstringUpdates generates update operations for the extracted substring.
// This function is exported for unit testing.
func BuildExtractSubstringUpdates(rows []map[string]interface{}, sourceColumnName string, generatedColumnName string, effectiveType string, req dto.ExtractSubstringRequest) ([]dto.UpdateColumnValueRequest, int, int) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	updated := 0
	skipped := 0
	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			skipped++
			continue
		}
		value, ok := row[sourceColumnName]
		if !ok || value == nil {
			skipped++
			continue
		}
		strVal, ok := value.(string)
		if !ok {
			skipped++
			continue
		}
		extracted, ok := ExtractSubstringByType(strVal, effectiveType, req.StartAfter, req.EndBefore)
		if !ok || strings.TrimSpace(extracted) == "" {
			skipped++
			continue
		}
		updates = append(updates, dto.UpdateColumnValueRequest{Id: rowID, Column: generatedColumnName, Value: extracted})
		updated++
	}
	return updates, updated, skipped
}

func (s tableManagementService) BuildExtractSubstringUpdates(rows []map[string]interface{}, sourceColumnName string, generatedColumnName string, effectiveType string, req dto.ExtractSubstringRequest) ([]dto.UpdateColumnValueRequest, int, int) {
	return BuildExtractSubstringUpdates(rows, sourceColumnName, generatedColumnName, effectiveType, req)
}

func DetermineMergeSeparator(format, custom string) string {
	switch format {
	case "space":
		return " "
	case "comma":
		return ", "
	case "dash":
		return "-"
	case "custom":
		return custom
	default:
		return " "
	}
}

func UniqueNameFromBase(baseName string, columnsData []dto.ColumnResponse) string {
	uniqueName := baseName
	suffix := 1
	for {
		exists := false
		for _, c := range columnsData {
			if c.ColumnName == uniqueName {
				exists = true
				break
			}
		}
		if !exists {
			break
		}
		uniqueName = fmt.Sprintf("%s_%d", baseName, suffix)
		suffix++
	}
	return uniqueName
}

func CombineColumnTitles(selectedColumns []string, columnsData []dto.ColumnResponse) string {
	titleMap := make(map[string]string, len(columnsData))
	for _, c := range columnsData {
		titleMap[c.ColumnName] = c.Title
	}
	parts := make([]string, 0, len(selectedColumns))
	for _, col := range selectedColumns {
		t := strings.TrimSpace(titleMap[col])
		if t == "" {
			t = col
		}
		parts = append(parts, t)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func UniqueTitleFromBase(base string, columnsData []dto.ColumnResponse) string {
	const maxTitleLen = 50
	baseTrim := strings.TrimSpace(base)
	if baseTrim == "" {
		return baseTrim
	}
	truncateRunes := func(s string, n int) string {
		r := []rune(strings.TrimSpace(s))
		if len(r) <= n {
			return string(r)
		}
		return string(r[:n])
	}
	titleExists := func(candidate string) bool {
		for _, c := range columnsData {
			if strings.EqualFold(strings.TrimSpace(c.Title), candidate) {
				return true
			}
		}
		return false
	}
	candidate := truncateRunes(baseTrim, maxTitleLen)
	if !titleExists(candidate) {
		return candidate
	}
	suffix := 2
	for {
		suffixStr := fmt.Sprintf(" %d", suffix)
		maxBase := maxTitleLen - len([]rune(suffixStr))
		if maxBase < 0 {
			maxBase = 0
		}
		basePart := truncateRunes(baseTrim, maxBase)
		candidate = basePart + suffixStr
		if !titleExists(candidate) {
			return candidate
		}
		suffix++
	}
}

func (s tableManagementService) DetermineDesiredOrderIndex(ctx context.Context, schemaName string, req dto.MergeColumnsRequest, columnsData []dto.ColumnResponse) (float64, error) {
	if req.AddAtEnd {
		maxOrder, err := s.columnsService.GetMaxOrderIndexOfColumn(ctx, schemaName, req.ModelID)
		if err != nil {
			return 0, err
		}
		return maxOrder + 1, nil
	}
	lastSel := strings.TrimSpace(req.Columns[len(req.Columns)-1])
	if idx, ok := FindLastSelectedOrderIndex(columnsData, lastSel); ok {
		desiredOrderIndex := idx + 1
		if err := s.ShiftColumnsStartingFrom(ctx, schemaName, desiredOrderIndex, columnsData); err != nil {
			return 0, err
		}
		return desiredOrderIndex, nil
	}
	maxOrder, err := s.columnsService.GetMaxOrderIndexOfColumn(ctx, schemaName, req.ModelID)
	if err != nil {
		return 0, err
	}
	return maxOrder + 1, nil
}

func FindLastSelectedOrderIndex(columnsData []dto.ColumnResponse, lastSel string) (float64, bool) {
	for _, c := range columnsData {
		if c.ID.String() == lastSel {
			if c.OrderIndex != nil {
				return *c.OrderIndex, true
			}
			return 0, true
		}
	}
	return 0, false
}

func (s tableManagementService) ShiftColumnsStartingFrom(ctx context.Context, schemaName string, start float64, columnsData []dto.ColumnResponse) error {
	for _, c := range columnsData {
		if c.OrderIndex != nil && *c.OrderIndex >= start {
			upd := dto.ColumnUpdate{OrderIndex: helpers.Float64Ptr(*c.OrderIndex + 1), UpdatedAt: time.Now().UTC()}
			if _, err := s.columnsService.UpdateColumn(ctx, schemaName, c.ID.String(), upd); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s tableManagementService) CreateNewColumnForMerge(ctx context.Context, schemaName string, model tenant.Model, req dto.MergeColumnsRequest, uniqueName, title string, desiredOrderIndex float64) (tenant.Column, error) {
	dt, err := s.getDataBaseType("longText")
	if err != nil {
		return tenant.Column{}, err
	}
	columnInsert := dto.ColumnInsertion{
		ID: uuid.New(), ModelID: uuid.MustParse(req.ModelID), BaseID: model.BaseID, Title: title, ColumnName: uniqueName,
		Description: helpers.StringPtr(""), Meta: map[string]interface{}{}, UIDT: "longText", DT: helpers.StringPtr(dt),
		Virtual: false, System: false, Deleted: false, OrderIndex: helpers.Float64Ptr(desiredOrderIndex), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	newCol, err := s.columnsService.Create(ctx, columnInsert, schemaName)
	if err != nil {
		return tenant.Column{}, err
	}
	if err := s.AddColumnInTableDb(schemaName, model.Alias, newCol); err != nil {
		return tenant.Column{}, err
	}
	return newCol, nil
}

func (s tableManagementService) BuildMergeUpdates(rows []map[string]interface{}, selectedColumns []string, sep, newColumnName string) ([]dto.UpdateColumnValueRequest, dto.MergeColumnsResponse) {
	result := dto.MergeColumnsResponse{TotalScanned: len(rows) * len(selectedColumns), TotalRows: len(rows)}
	if len(rows) == 0 {
		return nil, result
	}
	updates := make([]dto.UpdateColumnValueRequest, 0)
	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalRowsSkipped++
			result.TotalSkipped += len(selectedColumns)
			continue
		}
		tokens, skipped := CollectTokensFromRow(row, selectedColumns)
		result.TotalSkipped += skipped
		if len(tokens) == 0 {
			result.TotalRowsSkipped++
			continue
		}
		merged := strings.TrimSpace(strings.Join(tokens, sep))
		if merged == "" {
			result.TotalRowsSkipped++
			continue
		}
		updates = append(updates, dto.UpdateColumnValueRequest{Id: rowID, Column: newColumnName, Value: merged})
		result.TotalUpdated++
		result.TotalRowsUpdated++
	}
	return updates, result
}

func CollectTokensFromRow(row map[string]interface{}, selectedColumns []string) ([]string, int) {
	tokens := make([]string, 0, len(selectedColumns))
	skipped := 0
	for _, colName := range selectedColumns {
		value, exists := row[colName]
		if !exists || value == nil {
			skipped++
			continue
		}
		var strVal string
		switch v := value.(type) {
		case string:
			strVal = strings.TrimSpace(v)
		default:
			strVal = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if strVal == "" {
			skipped++
			continue
		}
		tokens = append(tokens, strVal)
	}
	return tokens, skipped
}

func (s tableManagementService) ApplyBulkUpdates(ctx context.Context, schemaName, modelAlias string, updates []dto.UpdateColumnValueRequest) error {
	if len(updates) == 0 {
		return nil
	}
	batchSize := columnActionBatchSize
	for start := 0; start < len(updates); start += batchSize {
		end := start + batchSize
		if end > len(updates) {
			end = len(updates)
		}
		if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, modelAlias, updates[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s tableManagementService) DeleteOriginalColumnsIfNeeded(ctx context.Context, schemaName string, req dto.MergeColumnsRequest, columnsData []dto.ColumnResponse) error {
	for _, colID := range req.Columns {
		colID = strings.TrimSpace(colID)
		for _, c := range columnsData {
			if c.ID.String() == colID {
				if err := s.DeleteColumnAndCleanUp(ctx, schemaName, c.ID.String(), c); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}
