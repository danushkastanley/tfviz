package input

import (
	"strings"
	"testing"
)

// FuzzReadErrorsNeverEchoInput plants a marker in arbitrary documents: the
// reader may reject them, but its message must never repeat input text.
func FuzzReadErrorsNeverEchoInput(f *testing.F) {
	for _, seed := range []string{
		`{"format_version":"MARKER","resource_changes":[]}`,
		`{"format_version":"1.2","resource_changes":[{"address":"MARKER","change":{"actions":"MARKER"}}]}`,
		`{"version":4,"lineage":"MARKER","terraform_version":"MARKER"}`,
		`{"@level":"info","@message":"MARKER"}`,
		`MARKER`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, doc string) {
		doc = strings.ReplaceAll(doc, "MARKER", "zq-SECRET-zq")
		for _, kind := range []SnapshotKind{KindPlan, KindState} {
			if _, err := Read([]byte(doc), kind); err != nil && strings.Contains(err.Error(), "SECRET") {
				t.Fatalf("error repeated input: %v", err)
			}
		}
	})
}
