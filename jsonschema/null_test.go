package jsonschema

import (
	"encoding/json"
	"testing"
)

func TestNull_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Null
		input   json.RawMessage
		wantErr bool
	}{
		{
			name:    "valid null",
			schema:  Null{},
			input:   json.RawMessage(`null`),
			wantErr: false,
		},
		{
			name:    "invalid boolean",
			schema:  Null{},
			input:   json.RawMessage(`true`),
			wantErr: true,
		},
		{
			name:    "invalid number",
			schema:  Null{},
			input:   json.RawMessage(`123`),
			wantErr: true,
		},
		{
			name:    "invalid string",
			schema:  Null{},
			input:   json.RawMessage(`"null"`),
			wantErr: true,
		},
		{
			name:    "invalid array",
			schema:  Null{},
			input:   json.RawMessage(`[]`),
			wantErr: true,
		},
		{
			name:    "invalid object",
			schema:  Null{},
			input:   json.RawMessage(`{}`),
			wantErr: true,
		},
		{
			name:    "invalid json",
			schema:  Null{},
			input:   json.RawMessage(`invalid`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Null.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNull_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		schema  Null
		want    string
		wantErr bool
	}{
		{
			name:    "empty schema",
			schema:  Null{},
			want:    `{"type":"null"}`,
			wantErr: false,
		},
		{
			name: "with description",
			schema: Null{
				Description: "test null",
			},
			want:    `{"type":"null","description":"test null"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.schema.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Null.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}
