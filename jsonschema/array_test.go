package jsonschema

import (
	"encoding/json"
	"testing"
)

func TestArray_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Array
		input   json.RawMessage
		wantErr bool
	}{
		{
			name: "valid array",
			schema: Array{
				Items: String{},
			},
			input:   json.RawMessage(`["value1", "value2"]`),
			wantErr: false,
		},
		{
			name: "invalid type",
			schema: Array{
				Items: String{},
			},
			input:   json.RawMessage(`"not an array"`),
			wantErr: true,
		},
		{
			name: "invalid item type",
			schema: Array{
				Items: String{},
			},
			input:   json.RawMessage(`["value1", 123]`),
			wantErr: true,
		},
		{
			name: "too few items",
			schema: Array{
				Items:    String{},
				MinItems: 3,
			},
			input:   json.RawMessage(`["value1", "value2"]`),
			wantErr: true,
		},
		{
			name: "too many items",
			schema: Array{
				Items:    String{},
				MaxItems: 1,
			},
			input:   json.RawMessage(`["value1", "value2"]`),
			wantErr: true,
		},
		{
			name: "nested array validation",
			schema: Array{
				Items: Array{
					Items: String{},
				},
			},
			input:   json.RawMessage(`[["inner1"], ["inner2"]]`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Array.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestArray_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		schema  Array
		want    string
		wantErr bool
	}{
		{
			name: "basic array",
			schema: Array{
				Description: "test array",
				Items:       String{},
				MinItems:    1,
				MaxItems:    10,
			},
			want:    `{"type":"array","description":"test array","items":{"type":"string"},"minItems":1,"maxItems":10}`,
			wantErr: false,
		},
		{
			name: "nested array",
			schema: Array{
				Items: Array{
					Items: String{},
				},
			},
			want:    `{"type":"array","items":{"type":"array","items":{"type":"string"}}}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.schema.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Array.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}
