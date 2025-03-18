package mcp

import (
	"reflect"
	"testing"
)

func TestResult_MarshalJSON(t *testing.T) {
	type Data struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}

	tests := []struct {
		name         string
		notification Result[Data]
		want         string
		wantErr      bool
	}{
		{
			name: "with meta",
			notification: Result[Data]{
				Meta: map[string]any{"meta1": "value1"},
				Data: Data{
					Field1: "value1",
					Field2: 2,
				},
			},
			want:    `{"field1":"value1","field2":2,"_meta":{"meta1":"value1"}}`,
			wantErr: false,
		},
		{
			name: "without meta",
			notification: Result[Data]{
				Data: Data{
					Field1: "value1",
					Field2: 2,
				},
			},
			want:    `{"field1":"value1","field2":2}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.notification.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}

func TestResult_UnmarshalJSON(t *testing.T) {
	type Data struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}

	tests := []struct {
		name    string
		data    string
		want    Result[Data]
		wantErr bool
	}{
		{
			name: "with meta",
			data: `{"field1":"value1","field2":2,"_meta":{"meta1":"value1"}}`,
			want: Result[Data]{
				Meta: map[string]any{"meta1": "value1"},
				Data: Data{
					Field1: "value1",
					Field2: 2,
				},
			},
			wantErr: false,
		},
		{
			name: "without meta",
			data: `{"field1":"value1","field2":2}`,
			want: Result[Data]{
				Data: Data{
					Field1: "value1",
					Field2: 2,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Result[Data]
			if err := got.UnmarshalJSON([]byte(tt.data)); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
