package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danushkastanley/tfviz/internal/testutil"
)

func TestIconsAreEmbeddedFromTheUsersPack(t *testing.T) {
	dir := t.TempDir()
	pack := testutil.RepoPath("testdata/icons/aws")
	for _, args := range [][]string{
		{"plan", "--input", producer("terraform-1.16/plan.json")},
		{"plan", "--input", producer("terraform-1.16/plan.json"), "--safe-share"},
		{"state", "--input", producer("terraform-1.16/prior.tfstate")},
	} {
		out := filepath.Join(dir, "report.html")
		res := run(t, nil, append(args, "--output", out, "--force", "--icons", pack)...)
		if res.code != ExitOK {
			t.Fatalf("%v: exit %d: %s", args, res.code, res.stderr)
		}
		if !strings.Contains(res.stderr, ". Icons for ") {
			t.Errorf("%v: summary = %q", args, res.stderr)
		}
		page, _ := os.ReadFile(out)
		for _, want := range []string{`"icons":{`, `"aws_nat_gateway":"ic`, `"images":{"ic1":"data:image/svg+xml;base64,`} {
			if !strings.Contains(string(page), want) {
				t.Errorf("%v: report lacks %s", args, want)
			}
		}
		testutil.AssertNoCanaries(t, "report", page)
	}
}

func TestReportsHaveNoIconsByDefault(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.html")
	res := run(t, nil, "plan", "--input", producer("terraform-1.16/plan.json"), "--output", out)
	page, _ := os.ReadFile(out)
	if res.code != ExitOK || strings.Contains(string(page), `"icons":`) || strings.Contains(res.stderr, ". Icons for") {
		t.Fatalf("unexpected icons: exit %d, %s", res.code, res.stderr)
	}
}

func TestUnusableIconFoldersAreUsageErrors(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.html")
	for _, dir := range []string{filepath.Join(t.TempDir(), "missing"), t.TempDir()} {
		res := run(t, nil, "plan", "--input", producer("terraform-1.16/plan.json"), "--output", out, "--icons", dir)
		if res.code != ExitUnsupported || !strings.Contains(res.stderr, "--icons:") {
			t.Errorf("%s: exit %d, %q", dir, res.code, res.stderr)
		}
		if _, err := os.Stat(out); err == nil {
			t.Fatal("a report was written despite the error")
		}
	}
}
