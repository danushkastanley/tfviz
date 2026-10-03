package aws

import (
	"flag"
	"os"
	"testing"

	"github.com/danushkastanley/tfviz/internal/testutil"
)

var update = flag.Bool("update", false, "rewrite docs/support.md")

func TestSupportDocumentIsCurrent(t *testing.T) {
	path := testutil.RepoPath("docs/support.md")
	want := SupportDocument()
	if *update {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatal("docs/support.md is out of date; run: go test ./internal/provider/aws -update")
	}
}
