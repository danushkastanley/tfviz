package projection

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// stateResource builds a state-mode resource from JSON via the real reader.
func stateResource(t *testing.T, values string) input.Resource {
	t.Helper()
	doc := `{"format_version":"1.0","values":{"root_module":{"resources":[{"address":"x_thing.a","mode":"managed","type":"x_thing","name":"a","provider_name":"registry.terraform.io/x/x","values":` + values + `,"sensitive_values":{}}]}}}`
	snap, err := input.Read([]byte(doc), input.KindState)
	if err != nil {
		t.Fatal(err)
	}
	return snap.Resources[0]
}

func after(t *testing.T, r input.Resource, f Field) string {
	t.Helper()
	got := Project(r, []Field{f}, false)
	data, _ := json.Marshal(got[0].After)
	return string(data)
}

func TestDenied(t *testing.T) {
	for _, key := range []string{"password", "master_password", "environment.variables", "user_data", "container_definitions", "policy", "a.b.private_key"} {
		if !Denied(key) {
			t.Errorf("%s should be denied", key)
		}
	}
	for _, key := range []string{"name", "client_authentication.sasl.scram", "kafka_version", "password_length_hint_x"} {
		if Denied(key) {
			t.Errorf("%s should be allowed", key)
		}
	}
}

func TestDeniedFieldsAreOmittedEvenIfApproved(t *testing.T) {
	r := stateResource(t, `{"password":"hunter2","user_data":"#!/bin/sh"}`)
	for _, f := range []Field{
		{Key: "password", Label: "Password"},
		{Key: "boot", Label: "Boot", Path: []string{"user_data"}},
		{Key: "summary", Label: "Summary", Inputs: []string{"user_data"}, Format: func(v input.Value) model.FieldValue {
			s, _ := v.Field("user_data").String()
			return model.KnownString(s)
		}},
	} {
		if got := after(t, r, f); got != `{"status":"omitted"}` {
			t.Errorf("%s exported %s", f.Key, got)
		}
	}
}

func TestStructuredValuesAreNeverDumped(t *testing.T) {
	r := stateResource(t, `{"config":{"nested":{"key":"value"}}}`)
	if got := after(t, r, Field{Key: "config", Label: "Config"}); got != `{"status":"omitted"}` {
		t.Fatalf("structured value without a formatter exported %s", got)
	}
}

func TestValuesAreBounded(t *testing.T) {
	long := strings.Repeat("é", 3000)
	items := make([]string, 400)
	for i := range items {
		items[i] = "s"
	}
	list, _ := json.Marshal(items)
	r := stateResource(t, `{"long":"`+long+`","many":`+string(list)+`}`)

	var text struct{ Value string }
	_ = json.Unmarshal([]byte(after(t, r, Field{Key: "long", Label: "Long"})), &text)
	if len(text.Value) > maxText || !strings.HasSuffix(text.Value, "…") || !json.Valid([]byte(`"`+text.Value+`"`)) {
		t.Fatalf("long text not clipped cleanly: %d bytes", len(text.Value))
	}
	var many struct{ Value []string }
	_ = json.Unmarshal([]byte(after(t, r, Field{Key: "many", Label: "Many"})), &many)
	if len(many.Value) != maxListItems || many.Value[maxListItems-1] != "…" {
		t.Fatalf("list not capped: %d items", len(many.Value))
	}
}

func TestWithheldFieldsAreOmitted(t *testing.T) {
	r := stateResource(t, `{"tags":{"Owner":"someone"}}`)
	if got := after(t, r, Field{Key: "tags", Label: "Tags", Withheld: true}); got != `{"status":"omitted"}` {
		t.Fatalf("withheld field exported %s", got)
	}
}
