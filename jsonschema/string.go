package jsonschema

import (
	"encoding/json"
	"fmt"
)

// String is a JSON schema string.
type String struct {
	Description string `json:"description,omitempty"`
	MinLength   int    `json:"minLength,omitempty"`
	MaxLength   int    `json:"maxLength,omitempty"`
}

// Validate validates the string against the JSON schema.
func (s String) Validate(v json.RawMessage) error {
	var m any
	if err := json.Unmarshal(v, &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return s.validate(m)
}

// validate validates the string against the JSON schema.
func (s String) validate(v any) error {
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
	type stringSchema String

	return json.Marshal(struct {
		Type string `json:"type"`
		stringSchema
	}{
		Type:         "string",
		stringSchema: stringSchema(s),
	})
}
