package jsonschema

import (
	"encoding/json"
	"fmt"
)

// String is a JSON schema string.
type String struct {
	MinLength int
	MaxLength int
}

// Validate validates the string against the JSON schema.
func (s *String) Validate(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("value is not a string")
	}

	if s.MinLength > 0 && len(str) < s.MinLength {
		return fmt.Errorf("string is too short")
	}

	if s.MaxLength > 0 && len(str) > s.MaxLength {
		return fmt.Errorf("string is too long")
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (s String) MarshalJSON() ([]byte, error) {
	obj := map[string]any{
		"type": "string",
	}

	if s.MinLength > 0 {
		obj["minLength"] = s.MinLength
	}

	if s.MaxLength > 0 {
		obj["maxLength"] = s.MaxLength
	}

	return json.Marshal(obj)
}
