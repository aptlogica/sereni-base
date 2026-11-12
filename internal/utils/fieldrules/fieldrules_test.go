// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package fieldrules

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromMetaReadsTheFieldSettings(t *testing.T) {
	// Checks follow the field's setting; without one, as the app's inputs: email and phone on, URL off
	assert.True(t, FromMeta("Email", nil).EmailCheck)
	assert.False(t, FromMeta("Email", map[string]interface{}{"emailValid": false}).EmailCheck)
	assert.True(t, FromMeta("PhoneNumber", nil).PhoneCheck)
	assert.False(t, FromMeta("PhoneNumber", map[string]interface{}{"phoneValid": false}).PhoneCheck)
	assert.False(t, FromMeta("URL", nil).URLCheck)
	assert.True(t, FromMeta("URL", map[string]interface{}{"urlValid": true}).URLCheck)

	assert.Equal(t, []string{"Open", "Done"}, FromMeta("Select", map[string]interface{}{
		"options": []interface{}{map[string]interface{}{"option": "Open"}, "Done"}}).Options)
	assert.Equal(t, 10, FromMeta("Rating", map[string]interface{}{"ratingMax": float64(10)}).RatingMax)
	assert.Equal(t, DefaultRatingMax, FromMeta("Rating", nil).RatingMax)

	// Decimal places: the app's pattern ("1.00") or a number; percent keeps 1
	assert.Equal(t, 2, FromMeta("Decimal", map[string]interface{}{"precision": "1.00"}).Places)
	assert.Equal(t, 3, FromMeta("Decimal", map[string]interface{}{"precision": float64(3)}).Places)
	assert.Equal(t, -1, FromMeta("Decimal", nil).Places)
	assert.Equal(t, 1, FromMeta("Percent", nil).Places)
	assert.Equal(t, -1, FromMeta("Currency", map[string]interface{}{"precision": "1.00"}).Places, "currency decimals are only shown")
}

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "first.last+tag@mail.example.com", "x_y%z@sub-domain.io"} {
		assert.True(t, ValidEmail(ok), ok)
	}
	for _, bad := range []string{"", "a@b", "a@b.c", "a b@c.com", "a@@b.com", "a@b.-co", "ä@b.com", "a@b_c.com", "a@.com"} {
		assert.False(t, ValidEmail(bad), bad)
	}
}

func TestValidPhone(t *testing.T) {
	for _, ok := range []string{"+91 98765 43210", "(22) 1234-5678", "9876543210", "+1-202-555-0143"} {
		assert.True(t, ValidPhone(ok), ok)
	}
	for _, bad := range []string{"", "0123456", "(020) 1234-5678", "+0123", "12345678901234567", "12.34.56", "abc", "+"} {
		assert.False(t, ValidPhone(bad), bad)
	}
}

func TestValidURL(t *testing.T) {
	for _, ok := range []string{"example.com", "https://example.com", "http://my-site.co.in/a/b_c", "docs.example.io/path to/file.pdf"} {
		assert.True(t, ValidURL(ok), ok)
	}
	for _, bad := range []string{"example", "https://Example.com", "example.c", "example.toolongtld", "example.com/a?b=1", "ex ample.com"} {
		assert.False(t, ValidURL(bad), bad)
	}
}

func TestParseTime(t *testing.T) {
	cases := map[string]string{"14:15": "14:15", "09:05": "09:05", "2:15 PM": "14:15", "12:00 am": "00:00", "12:30PM": "12:30", " 7:45 AM ": "07:45"}
	for in, want := range cases {
		got, ok := ParseTime(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"24:00", "9:5", "13:00 PM", "noon", ""} {
		_, ok := ParseTime(bad)
		assert.False(t, ok, bad)
	}
}

func TestFormatIssue(t *testing.T) {
	on := Rules{Type: TypeEmail, EmailCheck: true}
	assert.Equal(t, "must be a valid email address", FormatIssue(on, "nope"))
	assert.Empty(t, FormatIssue(on, ""), "an empty value is fine")
	assert.Empty(t, FormatIssue(Rules{Type: TypeEmail}, "nope"), "the field's check is off")
	assert.Equal(t, "must be a valid URL", FormatIssue(Rules{Type: TypeURL, URLCheck: true}, "nope"))
	assert.Equal(t, "must be a valid phone number", FormatIssue(Rules{Type: TypePhone, PhoneCheck: true}, "nope"))
}

func TestCheckCell(t *testing.T) {
	sel := Rules{Type: TypeSelect, Options: []string{"Open", "Done"}}
	v, issue := CheckCell(sel, "Open")
	assert.Equal(t, "Open", v)
	assert.Empty(t, issue)
	_, issue = CheckCell(sel, "") // the app clears a select with ""
	assert.Empty(t, issue)
	_, issue = CheckCell(sel, "Urgent")
	assert.Equal(t, `"Urgent" is not one of the field's options`, issue)
	_, issue = CheckCell(sel, float64(1))
	assert.NotEmpty(t, issue)

	multi := Rules{Type: TypeMultiSelect, Options: []string{"red", "blue"}}
	v, issue = CheckCell(multi, []interface{}{"red", "blue"})
	assert.Equal(t, []interface{}{"red", "blue"}, v)
	assert.Empty(t, issue)
	v, _ = CheckCell(multi, "red")
	assert.Equal(t, []interface{}{"red"}, v, "one option on its own is a list of one")
	v, _ = CheckCell(multi, "")
	assert.Equal(t, []interface{}{}, v)
	_, issue = CheckCell(multi, []interface{}{"red", "green"})
	assert.Equal(t, `"green" is not one of the field's options`, issue)

	_, issue = CheckCell(Rules{Type: TypeEmail, EmailCheck: true}, "nope")
	assert.Equal(t, "must be a valid email address", issue)
	v, issue = CheckCell(Rules{Type: TypeEmail}, "nope")
	assert.Equal(t, "nope", v, "the check is off")
	assert.Empty(t, issue)

	rating := Rules{Type: TypeRating, RatingMax: 5}
	_, issue = CheckCell(rating, float64(6))
	assert.Equal(t, "must be between 0 and 5", issue)
	_, issue = CheckCell(rating, float64(2.5))
	assert.Equal(t, "must be a whole number", issue)
	_, issue = CheckCell(rating, float64(3))
	assert.Empty(t, issue)

	v, _ = CheckCell(Rules{Type: TypeDecimal, Places: 2}, float64(1.236))
	assert.Equal(t, 1.24, v)
	v, _ = CheckCell(Rules{Type: TypeDecimal, Places: 2}, json.Number("2.005"))
	assert.InDelta(t, 2.01, v, 0.0001)
	v, _ = CheckCell(Rules{Type: TypePercent, Places: 1}, float64(12.34))
	assert.Equal(t, 12.3, v)
	v, _ = CheckCell(Rules{Type: TypeDecimal, Places: -1}, float64(1.23456))
	assert.Equal(t, 1.23456, v)

	v, _ = CheckCell(Rules{Type: TypeTime}, "2:15 PM")
	assert.Equal(t, "14:15", v)
	_, issue = CheckCell(Rules{Type: TypeTime}, "25:00")
	assert.NotEmpty(t, issue)

	// Other types pass through for their column type to check
	v, issue = CheckCell(Rules{Type: "number"}, "anything")
	assert.Equal(t, "anything", v)
	assert.Empty(t, issue)
}
