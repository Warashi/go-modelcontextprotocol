package jsonschema

import (
	"encoding/json"
	"testing"
)

func TestNumber_Validate(t *testing.T) {
	ptr := func(v float64) *float64 {
		return &v
	}

	tests := []struct {
		name    string
		schema  Number
		value   json.RawMessage
		wantErr bool
	}{
		{
			name:   "valid number",
			schema: Number{},
			value:  json.RawMessage(`42.5`),
		},
		{
			name: "number within range",
			schema: Number{
				Minimum: ptr(10.0),
				Maximum: ptr(100.0),
			},
			value: json.RawMessage(`50.0`),
		},
		{
			name: "number at minimum",
			schema: Number{
				Minimum: ptr(10.0),
			},
			value: json.RawMessage(`10.0`),
		},
		{
			name: "number at maximum",
			schema: Number{
				Maximum: ptr(100.0),
			},
			value: json.RawMessage(`100.0`),
		},
		{
			name: "number below minimum",
			schema: Number{
				Minimum: ptr(10.0),
			},
			value:   json.RawMessage(`5.0`),
			wantErr: true,
		},
		{
			name: "number above maximum",
			schema: Number{
				Maximum: ptr(100.0),
			},
			value:   json.RawMessage(`150.0`),
			wantErr: true,
		},
		{
			name: "number at exclusive minimum",
			schema: Number{
				ExclusiveMinimum: ptr(10.0),
			},
			value:   json.RawMessage(`10.0`),
			wantErr: true,
		},
		{
			name: "number at exclusive maximum",
			schema: Number{
				ExclusiveMaximum: ptr(100.0),
			},
			value:   json.RawMessage(`100.0`),
			wantErr: true,
		},
		{
			name:    "invalid json",
			schema:  Number{},
			value:   json.RawMessage(`invalid`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Number.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInteger_Validate(t *testing.T) {
	ptr := func(v int64) *int64 {
		return &v
	}

	tests := []struct {
		name    string
		schema  Integer
		value   json.RawMessage
		wantErr bool
	}{
		{
			name:   "valid integer",
			schema: Integer{},
			value:  json.RawMessage(`42`),
		},
		{
			name: "integer within range",
			schema: Integer{
				Minimum: ptr(int64(10)),
				Maximum: ptr(int64(100)),
			},
			value: json.RawMessage(`50`),
		},
		{
			name: "integer at minimum",
			schema: Integer{
				Minimum: ptr(int64(10)),
			},
			value: json.RawMessage(`10`),
		},
		{
			name: "integer at maximum",
			schema: Integer{
				Maximum: ptr(int64(100)),
			},
			value: json.RawMessage(`100`),
		},
		{
			name: "integer below minimum",
			schema: Integer{
				Minimum: ptr(int64(10)),
			},
			value:   json.RawMessage(`5`),
			wantErr: true,
		},
		{
			name: "integer above maximum",
			schema: Integer{
				Maximum: ptr(int64(100)),
			},
			value:   json.RawMessage(`150`),
			wantErr: true,
		},
		{
			name: "integer at exclusive minimum",
			schema: Integer{
				ExclusiveMinimum: ptr(int64(10)),
			},
			value:   json.RawMessage(`10`),
			wantErr: true,
		},
		{
			name: "integer at exclusive maximum",
			schema: Integer{
				ExclusiveMaximum: ptr(int64(100)),
			},
			value:   json.RawMessage(`100`),
			wantErr: true,
		},
		{
			name:    "non-integer number",
			schema:  Integer{},
			value:   json.RawMessage(`42.5`),
			wantErr: true,
		},
		{
			name:    "invalid json",
			schema:  Integer{},
			value:   json.RawMessage(`invalid`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Integer.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
