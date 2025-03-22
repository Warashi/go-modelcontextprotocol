package jsonschema

import (
	"encoding/json"
	"testing"
)

func TestConst_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Const
		input   string
		wantErr bool
	}{
		{
			name: "string const match",
			schema: Const{
				Value: "test",
			},
			input:   `"test"`,
			wantErr: false,
		},
		{
			name: "string const mismatch",
			schema: Const{
				Value: "test",
			},
			input:   `"other"`,
			wantErr: true,
		},
		{
			name: "number const match",
			schema: Const{
				Value: 42.0,
			},
			input:   `42`,
			wantErr: false,
		},
		{
			name: "number const mismatch",
			schema: Const{
				Value: 42.0,
			},
			input:   `43`,
			wantErr: true,
		},
		{
			name: "boolean const match",
			schema: Const{
				Value: true,
			},
			input:   `true`,
			wantErr: false,
		},
		{
			name: "boolean const mismatch",
			schema: Const{
				Value: true,
			},
			input:   `false`,
			wantErr: true,
		},
		{
			name: "null const match",
			schema: Const{
				Value: nil,
			},
			input:   `null`,
			wantErr: false,
		},
		{
			name: "null const mismatch",
			schema: Const{
				Value: nil,
			},
			input:   `42`,
			wantErr: true,
		},
		{
			name: "object const match",
			schema: Const{
				Value: map[string]any{"foo": "bar"},
			},
			input:   `{"foo":"bar"}`,
			wantErr: false,
		},
		{
			name: "object const mismatch",
			schema: Const{
				Value: map[string]any{"foo": "bar"},
			},
			input:   `{"foo":"baz"}`,
			wantErr: true,
		},
		{
			name: "array const match",
			schema: Const{
				Value: []any{1.0, 2.0, 3.0},
			},
			input:   `[1,2,3]`,
			wantErr: false,
		},
		{
			name: "array const mismatch",
			schema: Const{
				Value: []any{1.0, 2.0, 3.0},
			},
			input:   `[1,2,4]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(json.RawMessage(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Const.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConst_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		schema  Const
		want    string
		wantErr bool
	}{
		{
			name: "string const",
			schema: Const{
				Description: "test description",
				Value:       "test",
			},
			want:    `{"description":"test description","const":"test"}`,
			wantErr: false,
		},
		{
			name: "number const",
			schema: Const{
				Value: 42.0,
			},
			want:    `{"const":42}`,
			wantErr: false,
		},
		{
			name: "boolean const",
			schema: Const{
				Value: true,
			},
			want:    `{"const":true}`,
			wantErr: false,
		},
		{
			name: "null const",
			schema: Const{
				Value: nil,
			},
			want:    `{"const":null}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Const.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Const.MarshalJSON() = %v, want %v", string(got), tt.want)
			}
		})
	}
}
