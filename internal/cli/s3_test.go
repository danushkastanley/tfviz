package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danushkastanley/tfviz/internal/testutil"
)

// isolateAWS gives the SDK an empty environment with test keys, so the
// developer's real profiles and caches are never read.
func isolateAWS(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	for _, f := range []string{"config", "credentials"} {
		_ = os.WriteFile(filepath.Join(home, f), nil, 0o600)
	}
	for _, name := range []string{"AWS_PROFILE", "AWS_DEFAULT_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION", "AWS_SESSION_TOKEN", "AWS_ENDPOINT_URL", "AWS_ENDPOINT_URL_S3"} {
		t.Setenv(name, "")
	}
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(home, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(home, "credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
}

func runWithS3(t *testing.T, endpoint string, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, Env{
		Stdin: bytes.NewReader(nil), Stdout: &out, Stderr: &errOut, Version: "test", S3Endpoint: endpoint,
		Now: func() time.Time { return time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC) },
	})
	return result{code, out.String(), errOut.String()}
}

func TestStateFromS3(t *testing.T) {
	isolateAWS(t)
	state, _ := os.ReadFile(producer("terraform-1.16/prior.tfstate"))
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		w.Header().Set("Last-Modified", "Thu, 01 Oct 2026 08:30:00 GMT")
		w.Header().Set("x-amz-version-id", "3HL4kqtJlcpXroDTDmJ")
		_, _ = w.Write(state)
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "state.html")
	res := runWithS3(t, srv.URL, "state", "--input", "s3://state-bucket/prod/terraform.tfstate", "--aws-region", "eu-west-1", "--output", out)
	if res.code != ExitOK {
		t.Fatalf("exit %d: %s", res.code, res.stderr)
	}
	if len(methods) != 1 || methods[0] != "GET /state-bucket/prod/terraform.tfstate" {
		t.Fatalf("requests = %v", methods)
	}
	page, _ := os.ReadFile(out)
	for _, want := range []string{`"kind":"s3"`, `"timestamp":"2026-10-01T08:30:00Z"`, `"object_version":"3HL4kqtJlcpXroDTDmJ"`} {
		if !bytes.Contains(page, []byte(want)) {
			t.Errorf("report is missing %s", want)
		}
	}
	testutil.AssertNoCanaries(t, "S3 state report", page)
	testutil.AssertNoCanaries(t, "stderr", []byte(res.stderr))
}

func TestS3Failures(t *testing.T) {
	isolateAWS(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>m</Message></Error>`))
	}))
	defer srv.Close()
	dir := t.TempDir()
	plan := producer("terraform-1.16/plan.json")
	tests := []struct {
		name string
		args []string
		code int
		msg  string
	}{
		{"denied", []string{"state", "--input", "s3://state-bucket/k.tfstate", "--aws-region", "eu-west-1", "--output", filepath.Join(dir, "a.html")}, ExitFailure, "s3:GetObject"},
		{"offline", []string{"state", "--input", "s3://state-bucket/k.tfstate", "--offline", "--output", filepath.Join(dir, "b.html")}, ExitUnsupported, "--offline"},
		{"prefix", []string{"state", "--input", "s3://state-bucket/prefix/", "--output", filepath.Join(dir, "c.html")}, ExitUnsupported, "not a prefix"},
		{"s3 flag on local input", []string{"state", "--input", producer("terraform-1.16/state.json"), "--aws-profile", "work", "--output", filepath.Join(dir, "d.html")}, ExitUnsupported, "apply only to s3://"},
		{"s3 flag on plan", []string{"plan", "--input", plan, "--aws-profile", "work", "--output", filepath.Join(dir, "e.html")}, ExitUnsupported, "invalid options"},
		{"plan from s3", []string{"plan", "--input", "s3://state-bucket/plan.json", "--output", filepath.Join(dir, "f.html")}, ExitUnsupported, "show -json"},
		{"offline local file", []string{"plan", "--input", plan, "--offline", "--output", filepath.Join(dir, "g.html")}, ExitOK, "Wrote"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runWithS3(t, srv.URL, tt.args...)
			if res.code != tt.code || !strings.Contains(res.stderr, tt.msg) {
				t.Fatalf("exit %d (want %d): %s", res.code, tt.code, res.stderr)
			}
		})
	}
}
