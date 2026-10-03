package build

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
	"github.com/danushkastanley/tfviz/internal/testutil"
)

var update = flag.Bool("update", false, "rewrite golden report files")

var generatedAt = time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)

type fixture struct {
	name string
	path string
	kind input.SnapshotKind
}

var fixtures = []fixture{
	{"terraform-1.16-plan", "terraform-1.16/plan.json", input.KindPlan},
	{"terraform-1.16-state", "terraform-1.16/state.json", input.KindState},
	{"opentofu-1.13-plan", "opentofu-1.13/plan.json", input.KindPlan},
	{"opentofu-1.13-state", "opentofu-1.13/state.json", input.KindState},
}

func render(t *testing.T, f fixture) (*model.Report, []byte) {
	t.Helper()
	data, err := os.ReadFile(testutil.RepoPath("testdata/producer/" + f.path))
	if err != nil {
		t.Fatal(err)
	}
	snap, err := input.Read(data, f.kind)
	if err != nil {
		t.Fatal(err)
	}
	report := Report(snap, Options{Source: model.SourceStdin, GeneratedAt: generatedAt, ToolVersion: "test"})
	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return report, append(out, '\n')
}

// Golden files pin the complete report model for each fixture. Review the
// diff when they change; regenerate with: go test ./internal/report/build -update
func TestGoldenReports(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	schema, err := compiler.Compile(testutil.RepoPath("schema/report.v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			_, got := render(t, f)
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(got))
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(doc); err != nil {
				t.Fatalf("report does not match the schema: %v", err)
			}
			testutil.AssertNoCanaries(t, f.name+" report", got)

			golden := testutil.RepoPath("testdata/golden/" + f.name + ".json")
			if *update {
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden file (run with -update): %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("report differs from %s; review and run with -update if intended", golden)
			}
		})
	}
}

func TestReportIsDeterministic(t *testing.T) {
	_, a := render(t, fixtures[0])
	_, b := render(t, fixtures[0])
	if !bytes.Equal(a, b) {
		t.Fatal("the same input produced different reports")
	}
}

// Terraform and OpenTofu exports of the same stack must describe the same
// infrastructure; only producer details and completeness reporting differ.
func TestProducersAgree(t *testing.T) {
	for _, pair := range [][2]fixture{{fixtures[0], fixtures[2]}, {fixtures[1], fixtures[3]}} {
		tf, _ := render(t, pair[0])
		tofu, _ := render(t, pair[1])
		for _, r := range []*model.Report{tf, tofu} {
			r.Producer, r.Completeness, r.Warnings, r.Source = model.Producer{}, model.Completeness{}, nil, model.Source{}
		}
		a, _ := json.Marshal(tf)
		b, _ := json.Marshal(tofu)
		if !bytes.Equal(a, b) {
			t.Errorf("%s and %s describe different infrastructure", pair[0].name, pair[1].name)
		}
	}
}

func TestSummaryAndWarnings(t *testing.T) {
	plan, _ := render(t, fixtures[0])
	want := model.Summary{Create: 3, Update: 6, Delete: 1, Replace: 2, Read: 1, NoOp: 26}
	if plan.Summary != want {
		t.Errorf("summary = %+v, want %+v", plan.Summary, want)
	}
	if plan.Coverage != (model.Coverage{ResourcesTotal: 39, ResourcesSupported: 38, ResourcesGeneric: 1}) {
		t.Errorf("coverage = %+v", plan.Coverage)
	}
	if plan.Source.TimestampStatus != model.TimestampKnown || plan.Completeness.Status != model.CompletenessComplete {
		t.Errorf("source %+v completeness %+v", plan.Source, plan.Completeness)
	}
	tofu, _ := render(t, fixtures[2])
	if tofu.Completeness.Status != model.CompletenessNotReported {
		t.Errorf("OpenTofu completeness = %s", tofu.Completeness.Status)
	}
	found := false
	for _, w := range tofu.Warnings {
		found = found || w.Code == model.WarningCompletenessNotReported
	}
	if !found {
		t.Error("missing completeness warning for OpenTofu")
	}
}
