package jsonschema_test

import (
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/jsonschema"
)

func TestString_Validate(t *testing.T) {
	tests := []struct {
		name      string
		schema    jsonschema.String
		value     any
		expectErr bool
	}{
		{"valid string", jsonschema.String{MinLength: 3, MaxLength: 5}, "test", false},
		{"too short", jsonschema.String{MinLength: 3, MaxLength: 5}, "te", true},
		{"too long", jsonschema.String{MinLength: 3, MaxLength: 5}, "testing", true},
		{"not a string", jsonschema.String{MinLength: 3, MaxLength: 5}, 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.value)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func TestString_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		schema   jsonschema.String
		expected string
	}{
		{"min and max length", jsonschema.String{MinLength: 3, MaxLength: 5}, `{"maxLength":5,"minLength":3,"type":"string"}`},
		{"only min length", jsonschema.String{MinLength: 3}, `{"minLength":3,"type":"string"}`},
		{"only max length", jsonschema.String{MaxLength: 5}, `{"maxLength":5,"type":"string"}`},
		{"no constraints", jsonschema.String{}, `{"type":"string"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.schema.MarshalJSON()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, string(data))
			}
		})
	}
}
