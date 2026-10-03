package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danushkastanley/tfviz/internal/testutil"
)

type result struct {
	code           int
	stdout, stderr string
}

func run(t *testing.T, stdin []byte, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, Env{
		Stdin:   bytes.NewReader(stdin),
		Stdout:  &out,
		Stderr:  &errOut,
		Now:     func() time.Time { return time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC) },
		Version: "test",
	})
	return result{code, out.String(), errOut.String()}
}

func producer(path string) string { return testutil.RepoPath("testdata/producer/" + path) }

func TestGeneratesPlanAndStateReports(t *testing.T) {
	dir := t.TempDir()
	for _, tt := range []struct {
		command, input string
	}{
		{"plan", "terraform-1.16/plan.json"},
		{"plan", "opentofu-1.13/plan.json"},
		{"state", "terraform-1.16/state.json"},
		{"state", "opentofu-1.13/state.json"},
		{"state", "terraform-1.16/prior.tfstate"},
		{"state", "opentofu-1.13/prior.tfstate"},
	} {
		out := filepath.Join(dir, strings.ReplaceAll(tt.input, "/", "-")+".html")
		res := run(t, nil, tt.command, "--input", producer(tt.input), "--output", out)
		if res.code != ExitOK {
			t.Fatalf("%s %s: exit %d: %s", tt.command, tt.input, res.code, res.stderr)
		}
		info, err := os.Stat(out)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("report permissions = %v, want 0600", info.Mode().Perm())
		}
		page, _ := os.ReadFile(out)
		testutil.AssertNoCanaries(t, "report "+tt.input, page)
		testutil.AssertNoCanaries(t, "stdout", []byte(res.stdout))
		testutil.AssertNoCanaries(t, "stderr", []byte(res.stderr))
		if !strings.HasPrefix(res.stderr, "Wrote ") {
			t.Errorf("unexpected summary: %s", res.stderr)
		}
	}
}

func TestReadsStandardInput(t *testing.T) {
	data, _ := os.ReadFile(producer("terraform-1.16/plan.json"))
	out := filepath.Join(t.TempDir(), "report.html")
	res := run(t, data, "plan", "--input", "-", "--output", out, "--title", "Orders platform")
	if res.code != ExitOK {
		t.Fatalf("exit %d: %s", res.code, res.stderr)
	}
	page, _ := os.ReadFile(out)
	if !bytes.Contains(page, []byte("<title>Orders platform</title>")) || !bytes.Contains(page, []byte(`"kind":"stdin"`)) {
		t.Fatal("title or source kind missing")
	}
}

func TestExitCodes(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.html")
	_ = os.WriteFile(existing, []byte("keep me"), 0o600)
	plan := producer("terraform-1.16/plan.json")
	tests := []struct {
		name string
		args []string
		code int
		msg  string
	}{
		{"no command", nil, ExitUnsupported, "Usage"},
		{"unknown command", []string{"draw"}, ExitUnsupported, "unknown command"},
		{"unknown flag", []string{"plan", "--colour", "x"}, ExitUnsupported, "invalid options"},
		{"missing input", []string{"plan", "--output", filepath.Join(dir, "a.html")}, ExitUnsupported, "--input is required"},
		{"missing output", []string{"plan", "--input", plan}, ExitUnsupported, "--output is required"},
		{"stdout output", []string{"plan", "--input", plan, "--output", "-"}, ExitUnsupported, "must be a file path"},
		{"refuses overwrite", []string{"plan", "--input", plan, "--output", existing}, ExitUnsupported, "--force"},
		{"wrong kind", []string{"state", "--input", plan, "--output", filepath.Join(dir, "b.html")}, ExitUnsupported, "is a plan"},
		{"raw state as plan", []string{"plan", "--input", producer("terraform-1.16/prior.tfstate"), "--output", filepath.Join(dir, "c.html")}, ExitUnsupported, "not a plan"},
		{"missing file", []string{"plan", "--input", filepath.Join(dir, "nope.json"), "--output", filepath.Join(dir, "d.html")}, ExitFailure, "cannot read"},
		{"directory input", []string{"plan", "--input", dir, "--output", filepath.Join(dir, "e.html")}, ExitUnsupported, "regular file"},
		{"strict", []string{"plan", "--input", producer("terraform-1.16-platform/plan.json"), "--output", filepath.Join(dir, "f.html"), "--strict"}, ExitUnsupported, "unsupported resource"},
		{"strict passes when fully supported", []string{"plan", "--input", plan, "--output", filepath.Join(dir, "h.html"), "--strict"}, ExitOK, "Wrote"},
		{"bad view", []string{"plan", "--input", plan, "--output", filepath.Join(dir, "g.html"), "--view", "graph"}, ExitUnsupported, "--view"},
		{"inapplicable option", []string{"version", "--input", plan}, ExitOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := run(t, nil, tt.args...)
			if res.code != tt.code || !strings.Contains(res.stderr+res.stdout, tt.msg) {
				t.Fatalf("exit %d (want %d), output: %s%s", res.code, tt.code, res.stdout, res.stderr)
			}
		})
	}
	if kept, _ := os.ReadFile(existing); string(kept) != "keep me" {
		t.Fatal("an existing report was replaced without --force")
	}
	for _, name := range []string{"b.html", "c.html", "f.html"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Errorf("%s was written despite the failure", name)
		}
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, ".tfviz-*")); len(matches) > 0 {
		t.Errorf("temporary files left behind: %v", matches)
	}
}

func TestForceReplaces(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.html")
	_ = os.WriteFile(out, []byte("old"), 0o644)
	if res := run(t, nil, "plan", "--input", producer("terraform-1.16/plan.json"), "--output", out, "--force"); res.code != ExitOK {
		t.Fatalf("exit %d: %s", res.code, res.stderr)
	}
	info, _ := os.Stat(out)
	if info.Size() < 1000 || info.Mode().Perm() != 0o600 {
		t.Fatalf("replacement report: size %d perm %v", info.Size(), info.Mode().Perm())
	}
}

func TestErrorsNeverEchoInput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "r.html")
	res := run(t, []byte(`{"format_version":"CANARY-DB-PASSWORD-a41c"}`), "plan", "--input", "-", "--output", out)
	if res.code != ExitUnsupported {
		t.Fatalf("exit %d", res.code)
	}
	testutil.AssertNoCanaries(t, "stderr", []byte(res.stderr))
}

func TestVersionAndHelp(t *testing.T) {
	if res := run(t, nil, "version"); res.code != ExitOK || res.stdout != "tfviz test\n" {
		t.Fatalf("version: %+v", res)
	}
	if res := run(t, nil, "plan", "--help"); res.code != ExitOK || !strings.Contains(res.stderr, "-strict") {
		t.Fatalf("help: %+v", res)
	}
}
