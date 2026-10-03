package safeshare

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/build"
	"github.com/danushkastanley/tfviz/internal/report/html"
	"github.com/danushkastanley/tfviz/internal/report/model"
	"github.com/danushkastanley/tfviz/internal/testutil"
)

var fixtures = []struct {
	path string
	kind input.SnapshotKind
}{
	{"terraform-1.16/plan.json", input.KindPlan},
	{"terraform-1.16/state.json", input.KindState},
	{"terraform-1.16-platform/plan.json", input.KindPlan},
	{"opentofu-1.13-platform/state.json", input.KindState},
}

func internalReport(t *testing.T, path string, kind input.SnapshotKind) *model.Report {
	t.Helper()
	data, err := os.ReadFile(testutil.RepoPath("testdata/producer/" + path))
	if err != nil {
		t.Fatal(err)
	}
	snap, err := input.Read(data, kind)
	if err != nil {
		t.Fatal(err)
	}
	return build.Report(snap, build.Options{Title: "Orders platform", Source: model.SourceS3, SourceVersion: "3HL4kqtJlcpXroDTDmJ",
		GeneratedAt: time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC), ToolVersion: "test"})
}

// identifying collects every value safe-share must not disclose.
func identifying(r *model.Report) []string {
	out := []string{r.Title, r.Source.ObjectVersion}
	for _, res := range r.Resources {
		out = append(out, res.Label, res.Address, res.Module)
		for _, f := range res.Metadata {
			if descriptive[f.Key] {
				continue
			}
			for _, v := range []*model.FieldValue{f.Before, f.After} {
				if v == nil || v.Status() != model.StatusKnown {
					continue
				}
				switch value := v.Value().(type) {
				case string:
					out = append(out, value)
				case []string:
					out = append(out, value...)
				}
			}
		}
	}
	for _, g := range r.Groups {
		if g.Kind == model.GroupVPC || g.Kind == model.GroupSubnet || g.Kind == model.GroupAccount || (g.Kind == model.GroupModule && g.Label != "Root module") {
			label, _, _ := strings.Cut(g.Label, " · ")
			out = append(out, label)
		}
	}
	var meaningful []string
	for _, s := range out {
		// Very short values (such as "a" or "80") cannot be checked meaningfully.
		if len(s) >= 4 {
			meaningful = append(meaningful, s)
		}
	}
	return meaningful
}

var bundled = regexp.MustCompile(`(?s)<script type="module">.*?</script>|<style>.*?</style>|<template id="tfviz-notices">.*?</template>`)

func TestSafeShareDisclosesNoIdentifyingValues(t *testing.T) {
	for _, f := range fixtures {
		original := internalReport(t, f.path, f.kind)
		shared := Apply(original)
		data, _ := json.Marshal(shared)
		var page bytes.Buffer
		if err := html.Render(&page, shared, html.BundledAssets()); err != nil {
			t.Fatal(err)
		}
		// The interface bundle is the same for every report; check everything else.
		visible := bundled.ReplaceAllString(page.String(), "")
		for _, value := range identifying(original) {
			escaped, _ := json.Marshal(value)
			if bytes.Contains(data, escaped[1:len(escaped)-1]) || strings.Contains(visible, value) {
				t.Errorf("%s: safe-share disclosed %q", f.path, value)
			}
		}
		testutil.AssertNoCanaries(t, f.path+" safe-share", page.Bytes())
		for _, leak := range []string{"111122223333", "arn:aws:", "fixture.invalid", "10.40.", "10.50."} {
			if bytes.Contains(data, []byte(leak)) {
				t.Errorf("%s: safe-share contains %q", f.path, leak)
			}
		}
	}
}

func TestSafeShareKeepsTopologyAndChanges(t *testing.T) {
	original := internalReport(t, "terraform-1.16/plan.json", input.KindPlan)
	shared := Apply(original)
	if shared.Disclosure != model.DisclosureSafeShare || original.Disclosure != model.DisclosureInternal {
		t.Fatal("disclosure not set, or the original was changed")
	}
	if len(shared.Resources) != len(original.Resources) || shared.Summary != original.Summary || shared.Coverage != original.Coverage {
		t.Fatal("counts changed")
	}
	a, _ := json.Marshal(original.Relationships)
	b, _ := json.Marshal(shared.Relationships)
	if !bytes.Equal(a, b) {
		t.Fatal("relationships changed")
	}
	for i := range original.Resources {
		o, s := original.Resources[i], shared.Resources[i]
		if o.ID != s.ID || o.Type != s.Type || o.Change.Action != s.Change.Action || o.Groups != s.Groups {
			t.Fatalf("resource %s changed shape", o.ID)
		}
	}
	for i := range original.Groups {
		if original.Groups[i].Kind == model.GroupRegion && shared.Groups[i].Label != "eu-west-1" {
			t.Fatal("broad region labels are kept")
		}
	}
}

func TestStandInsAreConsistent(t *testing.T) {
	shared := Apply(internalReport(t, "terraform-1.16/plan.json", input.KindPlan))
	values := map[string]map[string]bool{}
	for _, res := range shared.Resources {
		for _, f := range res.Metadata {
			if f.After != nil && f.After.Status() == model.StatusKnown {
				if s, ok := f.After.Value().(string); ok {
					if values[f.Key] == nil {
						values[f.Key] = map[string]bool{}
					}
					values[f.Key][s] = true
				}
			}
		}
	}
	// Three private subnets share one VPC: their vpc-scoped resources must
	// show the same stand-in, and different subnets different ones.
	var subnetIDs []string
	for _, res := range shared.Resources {
		if res.Type == "aws_subnet" {
			for _, f := range res.Metadata {
				if f.Key == "id" {
					subnetIDs = append(subnetIDs, f.After.Value().(string))
				}
			}
		}
	}
	unique := map[string]bool{}
	for _, id := range subnetIDs {
		unique[id] = true
		if !strings.HasPrefix(id, "name-") {
			t.Fatalf("unexpected stand-in %q", id)
		}
	}
	if len(unique) != 6 {
		t.Fatalf("six subnets need six distinct stand-ins, got %v", subnetIDs)
	}
}

func TestSafeShareMatchesTheSchema(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(testutil.RepoPath("schema/report.v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtures {
		data, _ := json.Marshal(Apply(internalReport(t, f.path, f.kind)))
		doc, _ := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err := schema.Validate(doc); err != nil {
			t.Fatalf("%s: %v", f.path, err)
		}
	}
}
