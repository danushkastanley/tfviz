package source

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// isolate points the AWS SDK at an empty, temporary environment so tests
// never read the developer's real profiles, SSO cache or instance roles.
func isolate(t *testing.T, config string, withKeys bool) {
	t.Helper()
	home := t.TempDir()
	cfg := filepath.Join(home, "config")
	if err := os.WriteFile(cfg, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	creds := filepath.Join(home, "credentials")
	_ = os.WriteFile(creds, nil, 0o600)
	for _, name := range []string{"AWS_PROFILE", "AWS_DEFAULT_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION", "AWS_SESSION_TOKEN",
		"AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_ROLE_ARN", "AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "AWS_CONTAINER_CREDENTIALS_FULL_URI", "AWS_ENDPOINT_URL", "AWS_ENDPOINT_URL_S3"} {
		t.Setenv(name, "")
	}
	t.Setenv("HOME", home)
	t.Setenv("AWS_CONFIG_FILE", cfg)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", creds)
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	if withKeys {
		t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	} else {
		t.Setenv("AWS_ACCESS_KEY_ID", "")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	}
}

type fakeS3 struct {
	mu       sync.Mutex
	requests []*http.Request
}

func (f *fakeS3) start(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, r.Clone(context.Background()))
		f.mu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func s3Error(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>%s</Code><Message>m</Message></Error>`, code)
}

var loc = S3Location{Bucket: "state-bucket", Key: "prod/platform/terraform.tfstate"}

func TestFetchReadsExactlyOneObject(t *testing.T) {
	isolate(t, "", true)
	modified := time.Date(2026, 10, 1, 8, 30, 0, 0, time.UTC)
	f := &fakeS3{}
	endpoint := f.start(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", modified.Format(http.TimeFormat))
		w.Header().Set("x-amz-version-id", "v42")
		fmt.Fprint(w, `{"version":4}`)
	})
	obj, err := FetchS3(context.Background(), loc, S3Options{Region: "eu-west-1", VersionID: "v42", ExpectedBucketOwner: "111122223333", Endpoint: endpoint})
	if err != nil {
		t.Fatal(err)
	}
	if string(obj.Data) != `{"version":4}` || obj.VersionID != "v42" || obj.LastModified == nil || !obj.LastModified.Equal(modified) {
		t.Fatalf("object = %+v", obj)
	}
	if len(f.requests) != 1 {
		t.Fatalf("made %d requests, want exactly one", len(f.requests))
	}
	r := f.requests[0]
	if r.Method != http.MethodGet || r.URL.Path != "/state-bucket/prod/platform/terraform.tfstate" {
		t.Fatalf("request %s %s", r.Method, r.URL.Path)
	}
	if r.URL.Query().Get("versionId") != "v42" || r.Header.Get("X-Amz-Expected-Bucket-Owner") != "111122223333" {
		t.Fatalf("version or owner not sent: %v %v", r.URL.Query(), r.Header)
	}
}

func TestFetchExplainsFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		code   string
		opts   S3Options
		want   string
	}{
		{"denied", 403, "AccessDenied", S3Options{}, "s3:GetObject"},
		{"denied version", 403, "AccessDenied", S3Options{VersionID: "v1"}, "s3:GetObjectVersion"},
		{"denied owner", 403, "AccessDenied", S3Options{ExpectedBucketOwner: "1"}, "--expected-bucket-owner"},
		{"missing", 404, "NoSuchKey", S3Options{}, "does not exist"},
		{"wrong region", 301, "PermanentRedirect", S3Options{}, "--aws-region"},
		{"archived", 403, "InvalidObjectState", S3Options{}, "restored"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t, "", true)
			f := &fakeS3{}
			tt.opts.Endpoint = f.start(t, func(w http.ResponseWriter, r *http.Request) { s3Error(w, tt.status, tt.code) })
			tt.opts.Region = "eu-west-1"
			_, err := FetchS3(context.Background(), loc, tt.opts)
			var srcErr *Error
			if !errors.As(err, &srcErr) || !strings.Contains(srcErr.Message, tt.want) {
				t.Fatalf("got %v, want message containing %q", err, tt.want)
			}
			for _, r := range f.requests {
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected %s request", r.Method)
				}
			}
		})
	}
}

func TestFetchGuidesExpiredSSOSessions(t *testing.T) {
	isolate(t, "[profile work]\nsso_session = corp\nsso_account_id = 111122223333\nsso_role_name = ReadState\nregion = eu-west-1\n\n[sso-session corp]\nsso_start_url = https://example.invalid/start\nsso_region = eu-west-1\n", false)
	f := &fakeS3{}
	endpoint := f.start(t, func(w http.ResponseWriter, r *http.Request) { s3Error(w, 500, "Unexpected") })
	_, err := FetchS3(context.Background(), loc, S3Options{Profile: "work", Endpoint: endpoint})
	if err == nil || !strings.Contains(err.Error(), "aws sso login --profile work") {
		t.Fatalf("got %v", err)
	}
	if len(f.requests) != 0 {
		t.Fatal("no request may be sent without credentials")
	}
}

func TestFetchExplainsMissingCredentialsAndConfig(t *testing.T) {
	isolate(t, "", false)
	_, err := FetchS3(context.Background(), loc, S3Options{Region: "eu-west-1", Endpoint: "http://127.0.0.1:9"})
	if err == nil || !strings.Contains(err.Error(), "No usable AWS credentials") {
		t.Fatalf("got %v", err)
	}
	_, err = FetchS3(context.Background(), loc, S3Options{Profile: "missing"})
	if err == nil || !strings.Contains(err.Error(), `"missing" is not configured`) {
		t.Fatalf("got %v", err)
	}
	isolate(t, "", true)
	_, err = FetchS3(context.Background(), loc, S3Options{})
	if err == nil || !strings.Contains(err.Error(), "--aws-region") {
		t.Fatalf("got %v", err)
	}
}

func TestParseS3(t *testing.T) {
	good, err := ParseS3("s3://state-bucket/prod/a b/terraform.tfstate")
	if err != nil || good.Bucket != "state-bucket" || good.Key != "prod/a b/terraform.tfstate" {
		t.Fatalf("%+v %v", good, err)
	}
	for _, bad := range []string{"s3://", "s3://bucket", "s3://bucket/", "s3://Bucket/key", "s3://-bucket/key", "s3://a..b/key", "s3://bucket/prefix/", "https://bucket/key"} {
		if _, err := ParseS3(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}
