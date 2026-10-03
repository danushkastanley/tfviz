package build

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/html"
	"github.com/danushkastanley/tfviz/internal/report/model"
	"github.com/danushkastanley/tfviz/internal/report/safeshare"
	"github.com/danushkastanley/tfviz/internal/testutil"
)

var dataElement = regexp.MustCompile(`(?s)<script type="application/json" id="tfviz-report">(.*?)</script>`)

// FuzzPipeline drives arbitrary documents through the whole pipeline. Input
// is either rejected with an error or becomes a report that renders, embeds
// intact and survives safe-share; nothing may panic.
func FuzzPipeline(f *testing.F) {
	for _, path := range []string{"terraform-1.16/plan.json", "opentofu-1.13/state.json", "terraform-1.16-platform/prior.tfstate"} {
		data, err := os.ReadFile(testutil.RepoPath("testdata/producer/" + path))
		if err != nil {
			f.Fatal(err)
		}
		if len(data) > 64<<10 {
			data = data[:64<<10] // a truncated document is a useful seed too
		}
		f.Add(data)
	}
	f.Add(testutil.SyntheticPlan(40))
	f.Add([]byte(`{"format_version":"1.2","resource_changes":[{"address":"aws_vpc.x[\"</script><img src=x>\"]","mode":"managed","type":"aws_vpc","name":"x","provider_name":"registry.terraform.io/hashicorp/aws","change":{"actions":["create"],"before":null,"after":{"tags":{"Name":"</script> "}},"after_unknown":{"id":true}}}]}`))
	f.Add([]byte(`{"version":4,"lineage":"l","resources":[{"mode":"managed","type":"aws_subnet","name":"s","provider":"provider[\"registry.terraform.io/hashicorp/aws\"]","instances":[{"index_key":"a","attributes":{"id":"subnet-1"},"sensitive_attributes":[[{"type":"bogus"}]]}]}]}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, kind := range []input.SnapshotKind{input.KindPlan, input.KindState} {
			snap, err := input.Read(data, kind)
			if err != nil {
				continue
			}
			report := Report(snap, Options{Source: model.SourceFile, GeneratedAt: time.Unix(0, 0), ToolVersion: "fuzz"})
			for _, r := range []*model.Report{report, safeshare.Apply(report)} {
				var page bytes.Buffer
				if err := html.Render(&page, r, html.BundledAssets()); err != nil {
					t.Fatalf("render failed: %v", err)
				}
				m := dataElement.FindSubmatch(page.Bytes())
				if m == nil {
					t.Fatal("embedded report data is missing or was cut short")
				}
				var back model.Report
				if err := json.Unmarshal(m[1], &back); err != nil {
					t.Fatalf("embedded report data is not valid JSON: %v", err)
				}
				if len(back.Resources) != len(r.Resources) {
					t.Fatal("embedded report data lost resources")
				}
			}
		}
	})
}
