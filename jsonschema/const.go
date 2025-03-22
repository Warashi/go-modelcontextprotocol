package jsonschema

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Const is a JSON schema const.
type Const struct {
	Description string `json:"description,omitempty"`
	Value       any    `json:"const"`
}

// Validate validates the const against the JSON schema.
func (s Const) Validate(v json.RawMessage) error {
	var m any
	if err := json.Unmarshal(v, &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return s.validate(m)
}

// validate validates the const against the JSON schema.
func (s Const) validate(v any) error {
	if !reflect.DeepEqual(s.Value, v) {
		return fmt.Errorf("value does not match const value")
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (s Const) MarshalJSON() ([]byte, error) {
	type constSchema Const

	return json.Marshal(struct {
		constSchema
	}{
		constSchema: constSchema(s),
	})
}
