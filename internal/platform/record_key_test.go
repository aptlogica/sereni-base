// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skuRegionKey is a table whose primary key was changed to SKU + Region
func skuRegionKey() *recordKey {
	sku := &field{ID: "aaaaaaaa-0000-0000-0000-000000000001", Name: "SKU", ColumnName: "sku_x", Type: "singlelinetext"}
	region := &field{ID: "aaaaaaaa-0000-0000-0000-000000000002", Name: "Region", ColumnName: "region_x", Type: "singlelinetext"}
	size := &field{ID: "aaaaaaaa-0000-0000-0000-000000000003", Name: "Size", ColumnName: "size_x", Type: "singlelinetext"}
	tc := &tableContext{fields: []*field{sku, region, size}}
	return &recordKey{tc: tc, cols: []string{"sku_x", "region_x"}, fields: []*field{sku, region}}
}

func TestKeyObjectValues(t *testing.T) {
	k := skuRegionKey()
	// Any order, names or field IDs; "_" and other characters need no escaping
	values, apiErr := keyObjectValues(k, map[string]interface{}{"Region": "EU", "SKU": "A_1/2"})
	require.Nil(t, apiErr)
	assert.Equal(t, []string{"A_1/2", "EU"}, values)
	values, apiErr = keyObjectValues(k, map[string]interface{}{"fld_aaaaaaaa000000000000000000000001": json.Number("12"), "Region": "EU"})
	require.Nil(t, apiErr)
	assert.Equal(t, []string{"12", "EU"}, values)

	cases := map[string]map[string]interface{}{
		"The primary key is SKU + Region; Region is missing.":               {"SKU": "A1"},
		"Size is not part of the primary key (SKU + Region).":               {"SKU": "A1", "Region": "EU", "Size": "M"},
		"SKU can't be empty: it is part of the primary key (SKU + Region).": {"SKU": nil, "Region": "EU"},
		"SKU must be a single value.":                                       {"SKU": []interface{}{"A1"}, "Region": "EU"},
	}
	for want, key := range cases {
		_, apiErr := keyObjectValues(k, key)
		if assert.NotNil(t, apiErr, want) {
			assert.Equal(t, want, apiErr.Message)
		}
	}
	_, apiErr = keyObjectValues(k, map[string]interface{}{"Sku2": "A1"})
	assert.NotNil(t, apiErr, "an unknown field")
}

func TestKeyHints(t *testing.T) {
	k := skuRegionKey()
	assert.Equal(t, "records/by-key?key[SKU]=…&key[Region]=…", k.pathHint())
	assert.Equal(t, `{"SKU": …, "Region": …}`, k.objectHint())

	one := &recordKey{tc: k.tc, cols: []string{"sku_x"}, fields: []*field{k.fields[0]}}
	assert.Equal(t, "records/<SKU> or records/by-key?key[SKU]=…", one.pathHint())
}

func TestKeyFromQuery(t *testing.T) {
	k := skuRegionKey()
	key, apiErr := keyFromQuery(url.Values{"key[SKU]": {"A_1"}, "key[Region]": {"EU"}, "keys": {"id"}}, k)
	require.Nil(t, apiErr)
	assert.Equal(t, map[string]interface{}{"SKU": "A_1", "Region": "EU"}, key)

	_, apiErr = keyFromQuery(url.Values{}, k)
	require.NotNil(t, apiErr)
	assert.Equal(t, "Name the key fields: records/by-key?key[SKU]=…&key[Region]=….", apiErr.Message)
	_, apiErr = keyFromQuery(url.Values{"key[SKU]": {"A1", "A2"}}, k)
	assert.NotNil(t, apiErr)
}

func TestKeyValuesForResponses(t *testing.T) {
	k := skuRegionKey()
	row := map[string]interface{}{"id": json.Number("7"), "sku_x": "A1", "region_x": "EU", "size_x": "M"}
	b, err := json.Marshal(k.keyValues(row, fieldKeys(k.tc.fields, false)))
	require.NoError(t, err)
	assert.JSONEq(t, `{"SKU": "A1", "Region": "EU"}`, string(b))

	assert.Nil(t, (&recordKey{idKey: true}).keyValues(row, nil), "no key object with the default key")
	assert.Equal(t, `{"Region":"EU","SKU":"A1"}`, recordRef{key: map[string]interface{}{"SKU": "A1", "Region": "EU"}}.label())
}

func TestKeyWithTheRecordID(t *testing.T) {
	// A key of a UUID field plus the system id: id is named "id" and takes 12 or rec_12
	k := skuRegionKey()
	k.cols = []string{"sku_x", "id"}
	k.fields = []*field{k.fields[0], nil}
	for _, id := range []interface{}{"rec_12", "12", json.Number("12")} {
		values, apiErr := keyObjectValues(k, map[string]interface{}{"SKU": "A1", "id": id})
		require.Nil(t, apiErr, "%v", id)
		assert.Equal(t, []string{"A1", "12"}, values)
	}
	assert.Equal(t, "records/by-key?key[SKU]=…&key[id]=…", k.pathHint())
	_, apiErr := keyObjectValues(skuRegionKey(), map[string]interface{}{"SKU": "A1", "Region": "EU", "id": "1"})
	require.NotNil(t, apiErr)
	assert.Equal(t, "id is not part of the primary key (SKU + Region).", apiErr.Message)

	row := map[string]interface{}{"id": json.Number("12"), "sku_x": "A1"}
	b, err := json.Marshal(k.keyValues(row, fieldKeys(k.tc.fields, false)))
	require.NoError(t, err)
	assert.JSONEq(t, `{"SKU": "A1", "id": "rec_12"}`, string(b))
}
