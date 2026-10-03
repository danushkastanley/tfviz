package serve

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/build"
	"github.com/danushkastanley/tfviz/internal/report/html"
	"github.com/danushkastanley/tfviz/internal/report/model"
	"github.com/danushkastanley/tfviz/internal/testutil"
)

type harness struct {
	srv    *httptest.Server
	s      *Server
	loads  *atomic.Int32
	fail   *atomic.Bool
	client *http.Client
}

func start(t *testing.T) *harness {
	t.Helper()
	data, err := os.ReadFile(testutil.RepoPath("testdata/producer/terraform-1.16/plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	loads, fail := &atomic.Int32{}, &atomic.Bool{}
	load := func(context.Context) (*model.Report, error) {
		if fail.Load() {
			return nil, errors.New("the state object could not be retrieved")
		}
		n := loads.Add(1)
		snap, err := input.Read(data, input.KindPlan)
		if err != nil {
			return nil, err
		}
		return build.Report(snap, build.Options{Title: "load " + string(rune('0'+n)), Source: model.SourceFile, GeneratedAt: time.Unix(0, 0)}), nil
	}
	s, err := New(context.Background(), load, html.BundledAssets())
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	s.AllowHost(srv.Listener.Addr().String())
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &harness{srv, s, loads, fail, client}
}

func (h *harness) login(t *testing.T) {
	t.Helper()
	resp, err := h.client.Get(h.s.URL(h.srv.Listener.Addr().String()))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
		t.Fatalf("login: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	cookie := resp.Header.Get("Set-Cookie")
	if !strings.Contains(cookie, "HttpOnly") || !strings.Contains(cookie, "SameSite=Strict") {
		t.Fatalf("cookie attributes: %s", cookie)
	}
}

func (h *harness) get(t *testing.T, path string, header http.Header) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, h.srv.URL+path, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	if host := header.Get("Host"); host != "" {
		req.Host = host
	}
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

func TestPageNeedsTheLink(t *testing.T) {
	h := start(t)
	resp, body := h.get(t, "/", nil)
	if resp.StatusCode != http.StatusForbidden || strings.Contains(body, "tfviz-report") {
		t.Fatalf("unauthenticated: %d", resp.StatusCode)
	}
	resp, _ = h.get(t, "/?token=wrong", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong token: %d", resp.StatusCode)
	}
}

func TestPageAfterLogin(t *testing.T) {
	h := start(t)
	h.login(t)
	resp, body := h.get(t, "/", nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `id="tfviz-explore"`) || !strings.Contains(body, "connect-src &#39;self&#39;") {
		t.Fatalf("page: %d", resp.StatusCode)
	}
	for header, want := range map[string]string{"Cache-Control": "no-store", "X-Frame-Options": "DENY", "X-Content-Type-Options": "nosniff"} {
		if resp.Header.Get(header) != want {
			t.Errorf("%s = %q", header, resp.Header.Get(header))
		}
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("the explorer must not send CORS headers")
	}
	testutil.AssertNoCanaries(t, "explorer page", []byte(body))
}

func TestOtherHostsAreRefused(t *testing.T) {
	h := start(t)
	h.login(t)
	_, port, _ := net.SplitHostPort(h.srv.Listener.Addr().String())
	resp, body := h.get(t, "/", http.Header{"Host": {"attacker.example:" + port}})
	if resp.StatusCode != http.StatusMisdirectedRequest || strings.Contains(body, "tfviz-report") {
		t.Fatalf("DNS rebinding host: %d", resp.StatusCode)
	}
}

func (h *harness) refresh(t *testing.T, origin, token string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, h.srv.URL+refreshPath, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if token != "" {
		req.Header.Set(tokenHeader, token)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestRefreshIsExplicitAndSameOrigin(t *testing.T) {
	h := start(t)
	self := "http://" + h.srv.Listener.Addr().String()
	if code := h.refresh(t, self, h.s.csrf); code != http.StatusForbidden {
		t.Fatalf("refresh without the session cookie: %d", code)
	}
	h.login(t)
	for name, tt := range map[string]struct{ origin, token string }{
		"no token":     {self, ""},
		"wrong token":  {self, "nope"},
		"cross origin": {"https://attacker.example", h.s.csrf},
		"no origin":    {"", h.s.csrf},
	} {
		if code := h.refresh(t, tt.origin, tt.token); code != http.StatusForbidden {
			t.Errorf("%s: %d", name, code)
		}
	}
	if h.loads.Load() != 1 {
		t.Fatal("the snapshot was re-read without a valid refresh")
	}
	if code := h.refresh(t, self, h.s.csrf); code != http.StatusNoContent || h.loads.Load() != 2 {
		t.Fatalf("valid refresh: %d after %d loads", code, h.loads.Load())
	}
	if _, body := h.get(t, "/", nil); !strings.Contains(body, "<title>load 2</title>") {
		t.Fatal("the page was not rebuilt")
	}
	h.fail.Store(true)
	if code := h.refresh(t, self, h.s.csrf); code != http.StatusBadGateway {
		t.Fatalf("failed refresh: %d", code)
	}
	if _, body := h.get(t, "/", nil); !strings.Contains(body, "<title>load 2</title>") {
		t.Fatal("a failed refresh must keep the last good page")
	}
}

func TestUnknownRoutesAndMethods(t *testing.T) {
	h := start(t)
	h.login(t)
	if resp, _ := h.get(t, "/etc/passwd", nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown path: %d", resp.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodPut, h.srv.URL+"/", nil)
	resp, _ := h.client.Do(req)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("PUT: %d", resp.StatusCode)
	}
}

func TestListensOnLoopbackOnly(t *testing.T) {
	h := start(t)
	ln, err := h.s.Listen(0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)
	if !addr.IP.IsLoopback() {
		t.Fatalf("listening on %v", addr)
	}
	u, _ := url.Parse(h.s.URL(ln.Addr().String()))
	if u.Hostname() != "127.0.0.1" || u.Query().Get("token") == "" {
		t.Fatalf("URL = %s", u.Redacted())
	}
}
