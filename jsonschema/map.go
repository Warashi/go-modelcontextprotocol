package jsonschema

import (
	"encoding/json"
	"fmt"
)

// Map is a JSON schema object that maps keys to values.
//
// The additionalProperties field is a JSON schema that will be used to validate
// the values of the map.
type Map struct {
	Description          string `json:"description,omitempty"`
	AdditionalProperties Schema `json:"additionalProperties"`
}

func (s Map) Validate(v json.RawMessage) error {
	var m any
	if err := json.Unmarshal(v, &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return s.validate(m)
}

func (s Map) validate(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("value is not a object")
	}

	for k, v := range m {
		if err := s.AdditionalProperties.validate(v); err != nil {
			return fmt.Errorf("validate value %s: %w", k, err)
		}
	}

	return nil
}

func (s Map) MarshalJSON() ([]byte, error) {
	type mapSchema Map

	return json.Marshal(struct {
		Type string `json:"type"`
		mapSchema
	}{
		Type:      "object",
		mapSchema: mapSchema(s),
	})
}
