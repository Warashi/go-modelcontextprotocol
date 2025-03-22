package jsonschema

import (
	"encoding/json"
	"testing"
)

func TestMap_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Map
		input   json.RawMessage
		wantErr bool
	}{
		{
			name: "valid map",
			schema: Map{
				AdditionalProperties: String{},
			},
			input:   json.RawMessage(`{"key1": "value1", "key2": "value2"}`),
			wantErr: false,
		},
		{
			name: "invalid type",
			schema: Map{
				AdditionalProperties: String{},
			},
			input:   json.RawMessage(`"not a map"`),
			wantErr: true,
		},
		{
			name: "invalid value type",
			schema: Map{
				AdditionalProperties: String{},
			},
			input:   json.RawMessage(`{"key1": 123}`),
			wantErr: true,
		},
		{
			name: "nested map validation",
			schema: Map{
				AdditionalProperties: Map{
					AdditionalProperties: String{},
				},
			},
			input:   json.RawMessage(`{"outer": {"inner": "value"}}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Map.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMap_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		schema  Map
		want    string
		wantErr bool
	}{
		{
			name: "basic map",
			schema: Map{
				Description:          "test map",
				AdditionalProperties: String{},
			},
			want:    `{"type":"object","description":"test map","additionalProperties":{"type":"string"}}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.schema.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Map.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}
