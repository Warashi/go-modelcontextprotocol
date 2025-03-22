package jsonschema

import (
	"encoding/json"
	"fmt"
)

// Null is a JSON schema null.
type Null struct {
	Description string `json:"description,omitempty"`
}

// Validate validates the null against the JSON schema.
func (s Null) Validate(v json.RawMessage) error {
	var m any
	if err := json.Unmarshal(v, &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return s.validate(m)
}

// validate validates the null against the JSON schema.
func (s Null) validate(v any) error {
	if v != nil {
		return fmt.Errorf("value is not null")
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (s Null) MarshalJSON() ([]byte, error) {
	type nullSchema Null

	return json.Marshal(struct {
		Type string `json:"type"`
		nullSchema
	}{
		Type:       "null",
		nullSchema: nullSchema(s),
	})
}
