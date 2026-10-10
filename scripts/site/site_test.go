package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func built(t *testing.T) map[string][]byte {
	t.Helper()
	// Tests run in scripts/site; the generator works from the repo root.
	wd, _ := os.Getwd()
	if err := os.Chdir(filepath.Join("..", "..")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	files, err := build(config{Version: "0.0.0-test", Repo: "https://github.com/danushkastanley/tfviz"})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

var localRef = regexp.MustCompile(`(?:href|src|srcset)="(/[^"#?]*)`)

// Every local link and asset must exist in the published output.
func TestLocalLinksResolve(t *testing.T) {
	files := built(t)
	for name, data := range files {
		if !strings.HasSuffix(name, ".html") || strings.HasPrefix(name, "public/reports/") {
			continue
		}
		for _, m := range localRef.FindAllSubmatch(data, -1) {
			target := "public" + string(m[1])
			if strings.HasSuffix(target, "/") {
				target += "index.html"
			}
			if _, ok := files[target]; !ok {
				t.Errorf("%s links to %s, which is not published", name, m[1])
			}
		}
	}
}

// Pages run under a CSP with no inline scripts or styles.
func TestPagesNeedNoInlineScriptOrStyle(t *testing.T) {
	inline := regexp.MustCompile(`(?i)<script|\sstyle="|\son[a-z]+="|javascript:`)
	for name, data := range built(t) {
		if strings.HasSuffix(name, ".html") && !strings.HasPrefix(name, "public/reports/") {
			if loc := inline.FindIndex(data); loc != nil {
				t.Errorf("%s has inline script or style near %q", name, data[loc[0]:min(loc[1]+40, len(data))])
			}
		}
	}
}

// Every page gets the site CSP on both hosts, and demo reports never do.
func TestHeadersCoverEveryPageOnBothHosts(t *testing.T) {
	files := built(t)
	cf := string(files["public/_headers"])
	var vercel struct {
		Headers []struct {
			Source  string `json:"source"`
			Headers []struct{ Key, Value string }
		} `json:"headers"`
	}
	if err := json.Unmarshal(files["vercel.json"], &vercel); err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{}
	for _, h := range vercel.Headers {
		for _, kv := range h.Headers {
			if kv.Key == "Content-Security-Policy" {
				sources[h.Source] = kv.Value
			}
		}
	}
	for _, p := range pages {
		if !strings.Contains(cf, "\n"+p.Path+"\n  Content-Security-Policy: "+pageCSP+"\n") {
			t.Errorf("_headers lacks the page CSP for %s", p.Path)
		}
		if sources[p.Path] != pageCSP {
			t.Errorf("vercel.json lacks the page CSP for %s", p.Path)
		}
	}
	if sources["/reports/(.*)"] != demoCSP {
		t.Errorf("vercel.json reports CSP = %q", sources["/reports/(.*)"])
	}
	for _, p := range pages {
		if strings.HasPrefix(p.Path, "/reports/") {
			t.Errorf("page %s would receive both policies", p.Path)
		}
	}
}

func TestStylesheetIsFingerprinted(t *testing.T) {
	files := built(t)
	var css []string
	for name := range files {
		if strings.HasPrefix(name, "public/assets/site.") && strings.HasSuffix(name, ".css") {
			css = append(css, name)
		}
	}
	if len(css) != 1 {
		t.Fatalf("stylesheets = %v, want one fingerprinted file", css)
	}
	if !strings.Contains(string(files["public/index.html"]), strings.TrimPrefix(css[0], "public")) {
		t.Error("the home page does not reference the fingerprinted stylesheet")
	}
}
