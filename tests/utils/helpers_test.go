// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
package utils_test

import (
	"testing"

	"github.com/aptlogica/sereni-base/internal/utils/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapToStructWithValidData(t *testing.T) {
	type Person struct {
		Name string `mapstructure:"name"`
		Age  int    `mapstructure:"age"`
	}

	data := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
	}

	var result Person
	err := helpers.MapToStruct(data, &result)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", result.Name)
	assert.Equal(t, 30, result.Age)
}

func TestMapToStructWithPartialData(t *testing.T) {
	type Config struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}

	data := map[string]interface{}{
		"host": "localhost",
	}

	var result Config
	err := helpers.MapToStruct(data, &result)

	require.NoError(t, err)
	assert.Equal(t, "localhost", result.Host)
	assert.Equal(t, 0, result.Port)
}

func TestMapToStructWithTypeConversion(t *testing.T) {
	type Settings struct {
		Timeout int `mapstructure:"timeout"`
		Retry   int `mapstructure:"retry"`
	}

	data := map[string]interface{}{
		"timeout": "3000",
		"retry":   "5",
	}

	var result Settings
	err := helpers.MapToStruct(data, &result)

	require.Error(t, err)
}

func TestMapToStructWithNilFields(t *testing.T) {
	type Optional struct {
		Required string `mapstructure:"required"`
		Optional string `mapstructure:"optional"`
	}

	data := map[string]interface{}{
		"required": "value",
	}

	var result Optional
	err := helpers.MapToStruct(data, &result)

	require.NoError(t, err)
	assert.Equal(t, "value", result.Required)
	assert.Equal(t, "", result.Optional)
}

func TestMapToStructWithNestedData(t *testing.T) {
	type Address struct {
		City  string `mapstructure:"city"`
		State string `mapstructure:"state"`
	}

	type User struct {
		Name    string  `mapstructure:"name"`
		Address Address `mapstructure:"address"`
	}

	data := map[string]interface{}{
		"name": "Jane Smith",
		"address": map[string]interface{}{
			"city":  "New York",
			"state": "NY",
		},
	}

	var result User
	err := helpers.MapToStruct(data, &result)

	require.NoError(t, err)
	assert.Equal(t, "Jane Smith", result.Name)
	assert.Equal(t, "New York", result.Address.City)
	assert.Equal(t, "NY", result.Address.State)
}

func TestMapToStructWithInvalidNilPointer(t *testing.T) {
	data := map[string]interface{}{
		"name": "test",
	}

	err := helpers.MapToStruct(data, nil)
	assert.Error(t, err)
}

func TestMapToStructWithEmptyMap(t *testing.T) {
	type Empty struct {
		Value string
	}

	data := map[string]interface{}{}
	var result Empty

	err := helpers.MapToStruct(data, &result)
	require.NoError(t, err)
	assert.Equal(t, "", result.Value)
}

func TestMapToStructWithBoolConversion(t *testing.T) {
	type Status struct {
		Active bool `mapstructure:"active"`
	}

	data := map[string]interface{}{
		"active": true,
	}

	var result Status
	err := helpers.MapToStruct(data, &result)

	require.NoError(t, err)
	assert.True(t, result.Active)
}

func TestMapToStructWithFloatConversion(t *testing.T) {
	type Measurement struct {
		Value float64 `mapstructure:"value"`
	}

	data := map[string]interface{}{
		"value": 3.14,
	}

	var result Measurement
	err := helpers.MapToStruct(data, &result)

	require.NoError(t, err)
	assert.Equal(t, 3.14, result.Value)
}
