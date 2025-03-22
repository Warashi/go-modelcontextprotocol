package jsonschema

import (
	"encoding/json"
	"fmt"
)

// Array is a JSON schema array.
type Array struct {
	Description string `json:"description,omitempty"`
	MinItems    int    `json:"minItems,omitempty"`
	MaxItems    int    `json:"maxItems,omitempty"`
	Items       Schema `json:"items"`
}

// Validate validates the array against the JSON schema.
func (s Array) Validate(v json.RawMessage) error {
	var m any
	if err := json.Unmarshal(v, &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return s.validate(m)
}

// validate validates the array against the JSON schema.
func (s Array) validate(v any) error {
	arr, ok := v.([]any)
	if !ok {
		return fmt.Errorf("value is not an array")
	}

	if s.MinItems > 0 && len(arr) < s.MinItems {
		return fmt.Errorf("array has too few items")
	}

	if s.MaxItems > 0 && len(arr) > s.MaxItems {
		return fmt.Errorf("array has too many items")
	}

	for i, v := range arr {
		if err := s.Items.validate(v); err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (s Array) MarshalJSON() ([]byte, error) {
	type arraySchema Array

	return json.Marshal(struct {
		Type string `json:"type"`
		arraySchema
	}{
		Type:        "array",
		arraySchema: arraySchema(s),
	})
}
