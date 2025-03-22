package jsonschema

import (
	"encoding/json"
	"fmt"
)

// Boolean is a JSON schema boolean.
type Boolean struct {
	Description string `json:"description,omitempty"`
}

// Validate validates the boolean against the JSON schema.
func (s Boolean) Validate(v json.RawMessage) error {
	var m any
	if err := json.Unmarshal(v, &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return s.validate(m)
}

// validate validates the boolean against the JSON schema.
func (s Boolean) validate(v any) error {
	_, ok := v.(bool)
	if !ok {
		return fmt.Errorf("value is not a boolean")
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (s Boolean) MarshalJSON() ([]byte, error) {
	type booleanSchema Boolean

	return json.Marshal(struct {
		Type string `json:"type"`
		booleanSchema
	}{
		Type:          "boolean",
		booleanSchema: booleanSchema(s),
	})
}
