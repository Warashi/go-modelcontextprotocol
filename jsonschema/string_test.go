package jsonschema

import (
	"testing"
)

func TestString_Validate(t *testing.T) {
	tests := []struct {
		name      string
		schema    String
		value     any
		expectErr bool
	}{
		{"valid string", String{MinLength: 3, MaxLength: 5}, "test", false},
		{"too short", String{MinLength: 3, MaxLength: 5}, "te", true},
		{"too long", String{MinLength: 3, MaxLength: 5}, "testing", true},
		{"not a string", String{MinLength: 3, MaxLength: 5}, 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.validate(tt.value)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func TestString_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		schema   String
		expected string
	}{
		{"min and max length", String{MinLength: 3, MaxLength: 5}, `{"maxLength":5,"minLength":3,"type":"string"}`},
		{"only min length", String{MinLength: 3}, `{"minLength":3,"type":"string"}`},
		{"only max length", String{MaxLength: 5}, `{"maxLength":5,"type":"string"}`},
		{"no constraints", String{}, `{"type":"string"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.schema.MarshalJSON()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.expected, string(data))
		})
	}
}
