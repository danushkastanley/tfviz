package model_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/danushkastanley/tfviz/internal/report/model"
)

func repoPath(t *testing.T, rel string) string {
	t.Helper()
	return filepath.Join("..", "..", "..", rel)
}

func compileSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	schema, err := compiler.Compile(repoPath(t, "schema/report.v1.schema.json"))
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return schema
}

func validate(t *testing.T, schema *jsonschema.Schema, data []byte) error {
	t.Helper()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	return schema.Validate(doc)
}

func TestSampleReportMatchesSchema(t *testing.T) {
	data, err := os.ReadFile(repoPath(t, "testdata/reports/aws-review.sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validate(t, compileSchema(t), data); err != nil {
		t.Fatalf("sample report does not match schema: %v", err)
	}
}

// The Go types must round-trip the sample without loss and still validate,
// which keeps the hand-written structs aligned with the schema.
func TestGoTypesRoundTripSample(t *testing.T) {
	original, err := os.ReadFile(repoPath(t, "testdata/reports/aws-review.sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(original))
	dec.DisallowUnknownFields()
	var report model.Report
	if err := dec.Decode(&report); err != nil {
		t.Fatalf("decode into model.Report: %v", err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := validate(t, compileSchema(t), encoded); err != nil {
		t.Fatalf("re-encoded report does not match schema: %v", err)
	}
	var want, got any
	_ = json.Unmarshal(original, &want)
	_ = json.Unmarshal(encoded, &got)
	wantJSON, _ := json.Marshal(want)
	gotJSON, _ := json.Marshal(got)
	if !bytes.Equal(wantJSON, gotJSON) {
		t.Fatal("round trip changed the report")
	}
}

func TestSchemaRejectsSensitivePayload(t *testing.T) {
	schema := compileSchema(t)
	data, err := os.ReadFile(repoPath(t, "testdata/reports/aws-review.sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	resource := doc["resources"].([]any)[0].(map[string]any)
	resource["metadata"] = []any{map[string]any{
		"key": "password", "label": "Password",
		"after": map[string]any{"status": "sensitive", "value": "leak"},
	}}
	tampered, _ := json.Marshal(doc)
	if err := validate(t, schema, tampered); err == nil {
		t.Fatal("schema accepted a sensitive field carrying a value")
	}

	resource["metadata"] = []any{}
	resource["attributes"] = map[string]any{"password": "leak"}
	tampered, _ = json.Marshal(doc)
	if err := validate(t, schema, tampered); err == nil {
		t.Fatal("schema accepted a generic attribute map")
	}
}
