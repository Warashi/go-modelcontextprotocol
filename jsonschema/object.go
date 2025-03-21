package jsonschema

import (
	"encoding/json"
	"fmt"
)

// Object is a JSON schema object.
type Object struct {
	Properties map[string]Schema
	Required   []string
}

// MarshalJSON implements the json.Marshaler interface.
func (o Object) MarshalJSON() ([]byte, error) {
	for _, r := range o.Required {
		if _, ok := o.Properties[r]; !ok {
			return nil, fmt.Errorf("required property %s not found", r)
		}
	}

	obj := map[string]any{
		"type":                 "object",
		"additionalProperties": false, // we set this to false for simplicity
	}

	if len(o.Required) > 0 {
		obj["required"] = o.Required
	}

	if len(o.Properties) > 0 {
		properties := make(map[string]any)
		for k, v := range o.Properties {
			properties[k] = v
		}
		obj["properties"] = properties
	}

	return json.Marshal(obj)
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
