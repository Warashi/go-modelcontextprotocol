package jsonschema

import (
	"encoding/json"
	"testing"
)

func TestBoolean_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Boolean
		input   json.RawMessage
		wantErr bool
	}{
		{
			name:    "valid true",
			schema:  Boolean{},
			input:   json.RawMessage(`true`),
			wantErr: false,
		},
		{
			name:    "valid false",
			schema:  Boolean{},
			input:   json.RawMessage(`false`),
			wantErr: false,
		},
		{
			name:    "invalid number",
			schema:  Boolean{},
			input:   json.RawMessage(`123`),
			wantErr: true,
		},
		{
			name:    "invalid string",
			schema:  Boolean{},
			input:   json.RawMessage(`"true"`),
			wantErr: true,
		},
		{
			name:    "invalid null",
			schema:  Boolean{},
			input:   json.RawMessage(`null`),
			wantErr: true,
		},
		{
			name:    "invalid json",
			schema:  Boolean{},
			input:   json.RawMessage(`invalid`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Boolean.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBoolean_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		schema  Boolean
		want    string
		wantErr bool
	}{
		{
			name:    "empty schema",
			schema:  Boolean{},
			want:    `{"type":"boolean"}`,
			wantErr: false,
		},
		{
			name: "with description",
			schema: Boolean{
				Description: "test boolean",
			},
			want:    `{"type":"boolean","description":"test boolean"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.schema.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Boolean.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}
