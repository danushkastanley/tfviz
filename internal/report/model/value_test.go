package model

import (
	"encoding/json"
	"testing"
)

func TestFieldValueMarshalling(t *testing.T) {
	tests := []struct {
		name  string
		value FieldValue
		want  string
	}{
		{"known false is kept", KnownBool(false), `{"status":"known","value":false}`},
		{"known zero is kept", KnownNumber(0), `{"status":"known","value":0}`},
		{"known empty string is kept", KnownString(""), `{"status":"known","value":""}`},
		{"known list", KnownList([]string{"a", "b"}), `{"status":"known","value":["a","b"]}`},
		{"sensitive has no payload", Sensitive(), `{"status":"sensitive"}`},
		{"unknown has no payload", Unknown(), `{"status":"unknown"}`},
		{"omitted has no payload", Omitted(), `{"status":"omitted"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFieldValueRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{
		`{"status":"sensitive","value":"leak"}`,
		`{"status":"known"}`,
		`{"status":"known","value":null}`,
		`{"status":"known","value":{"nested":"object"}}`,
		`{"status":"masked"}`,
	} {
		var v FieldValue
		if err := json.Unmarshal([]byte(input), &v); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}

func TestZeroFieldValueCannotMarshal(t *testing.T) {
	if _, err := json.Marshal(FieldValue{}); err == nil {
		t.Fatal("marshalled a FieldValue with no status")
	}
}
