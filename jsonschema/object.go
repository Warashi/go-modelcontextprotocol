package jsonschema

import (
	"encoding/json"
	"fmt"
)

// Object is a JSON schema object.
type Object struct {
	Description string            `json:"description,omitempty"`
	Properties  map[string]Schema `json:"properties"`
	Required    []string          `json:"required,omitempty,omitzero"`
}

// MarshalJSON implements the json.Marshaler interface.
func (o Object) MarshalJSON() ([]byte, error) {
	for _, r := range o.Required {
		if _, ok := o.Properties[r]; !ok {
			return nil, fmt.Errorf("required property %s not found", r)
		}
	}

	type objectSchema Object

	return json.Marshal(struct {
		Type                 string `json:"type"`
		AdditionalProperties bool   `json:"additionalProperties"`
		objectSchema
	}{
		Type:                 "object",
		AdditionalProperties: false, // we set this to false for simplicity
		objectSchema:         objectSchema(o),
	})
}

// Validate validates the object against the JSON schema.
func (o Object) Validate(v json.RawMessage) error {
	var m any
	if err := json.Unmarshal(v, &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return o.validate(m)
}

// validate validates the object against the JSON schema.
func (o Object) validate(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("object is not a map")
	}

	for _, r := range o.Required {
		if _, ok := m[r]; !ok {
			return fmt.Errorf("required property %s not found", r)
		}
	}

	for k := range m {
		if _, ok := o.Properties[k]; !ok {
			return fmt.Errorf("unexpected property %s", k)
		}
	}

	for k, v := range o.Properties {
		if err := v.validate(m[k]); err != nil {
			return fmt.Errorf("property %s: %w", k, err)
		}
	}

	return nil
}
