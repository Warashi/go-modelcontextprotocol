package jsonschema

import (
	"encoding/json"
	"fmt"
)

// Number is a JSON schema for a number.
type Number struct {
	Description      string   `json:"description,omitempty"`
	Minimum          *float64 `json:"minimum,omitempty,omitzero"`
	Maximum          *float64 `json:"maximum,omitempty,omitzero"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty,omitzero"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty,omitzero"`
}

func (s Number) Validate(v json.RawMessage) error {
	var n any
	if err := json.Unmarshal(v, &n); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return s.validate(n)
}

func (s Number) validate(v any) error {
	n, ok := v.(float64)
	if !ok {
		return fmt.Errorf("value is not a number")
	}

	if s.Minimum != nil && n < *s.Minimum {
		return fmt.Errorf("number is less than minimum")
	}

	if s.Maximum != nil && n > *s.Maximum {
		return fmt.Errorf("number is greater than maximum")
	}

	if s.ExclusiveMinimum != nil && n <= *s.ExclusiveMinimum {
		return fmt.Errorf("number is less than or equal to exclusive minimum")
	}

	if s.ExclusiveMaximum != nil && n >= *s.ExclusiveMaximum {
		return fmt.Errorf("number is greater than or equal to exclusive maximum")
	}

	return nil
}

func (s Number) MarshalJSON() ([]byte, error) {
	type numberSchema Number

	return json.Marshal(struct {
		Type string `json:"type"`
		numberSchema
	}{
		Type:         "number",
		numberSchema: numberSchema(s),
	})
}

// Integer is a JSON schema for an integer.
type Integer struct {
	Description      string `json:"description,omitempty"`
	Minimum          *int64 `json:"minimum,omitempty,omitzero"`
	Maximum          *int64 `json:"maximum,omitempty,omitzero"`
	ExclusiveMinimum *int64 `json:"exclusiveMinimum,omitempty,omitzero"`
	ExclusiveMaximum *int64 `json:"exclusiveMaximum,omitempty,omitzero"`
}

func (s Integer) Validate(v json.RawMessage) error {
	var n any
	if err := json.Unmarshal(v, &n); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return s.validate(n)
}

func (s Integer) validate(v any) error {
	n, ok := v.(float64)
	if !ok {
		return fmt.Errorf("value is not a number")
	}

	if float64(int64(n)) != n {
		return fmt.Errorf("value is not an integer")
	}

	i := int64(n)

	if s.Minimum != nil && i < *s.Minimum {
		return fmt.Errorf("number is less than minimum")
	}

	if s.Maximum != nil && i > *s.Maximum {
		return fmt.Errorf("number is greater than maximum")
	}

	if s.ExclusiveMinimum != nil && i <= *s.ExclusiveMinimum {
		return fmt.Errorf("number is less than or equal to exclusive minimum")
	}

	if s.ExclusiveMaximum != nil && i >= *s.ExclusiveMaximum {
		return fmt.Errorf("number is greater than or equal to exclusive maximum")
	}

	return nil
}

func (s Integer) MarshalJSON() ([]byte, error) {
	type integerSchema Integer

	return json.Marshal(struct {
		Type string `json:"type"`
		integerSchema
	}{
		Type:          "integer",
		integerSchema: integerSchema(s),
	})
}
