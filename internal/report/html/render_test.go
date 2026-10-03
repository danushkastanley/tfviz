package html

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/danushkastanley/tfviz/internal/report/model"
)

func loadSample(t *testing.T) *model.Report {
	t.Helper()
	raw, err := os.ReadFile("../../../testdata/reports/aws-review.sample.json")
	if err != nil {
		t.Fatal(err)
	}
	var report model.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	return &report
}

func render(t *testing.T, report *model.Report) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, report, BundledAssets()); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

var inlineElement = regexp.MustCompile(`(?s)<(script|style)( type="module")?>(.*?)</(?:script|style)>`)

func TestCSPHashesMatchInlinedContent(t *testing.T) {
	page := render(t, loadSample(t))
	matches := inlineElement.FindAllStringSubmatch(page, -1)
	if len(matches) != 2 {
		t.Fatalf("expected one inline script and one stylesheet, found %d", len(matches))
	}
	for _, m := range matches {
		sum := sha256.Sum256([]byte(m[3]))
		want := "sha256-" + base64.StdEncoding.EncodeToString(sum[:])
		if !strings.Contains(page, want) {
			t.Errorf("CSP does not allow the inlined %s (missing %s)", m[1], want)
		}
	}
}

func TestCSPDeniesNetworkAndEval(t *testing.T) {
	page := render(t, loadSample(t))
	for _, required := range []string{"default-src &#39;none&#39;", "connect-src &#39;none&#39;", "base-uri &#39;none&#39;", "form-action &#39;none&#39;"} {
		if !strings.Contains(page, required) {
			t.Errorf("CSP is missing %q", required)
		}
	}
	for _, forbidden := range []string{"unsafe-eval", "unsafe-inline", "http://", "https://"} {
		csp := page[strings.Index(page, "Content-Security-Policy"):]
		csp = csp[:strings.Index(csp, ">")]
		if strings.Contains(csp, forbidden) {
			t.Errorf("CSP contains %q", forbidden)
		}
	}
}

func TestHostileLabelsCannotEscapeTheirContext(t *testing.T) {
	// The bundled interface itself contains strings such as "<script", so
	// compare markup counts with a render of the untouched sample.
	baseline := render(t, loadSample(t))
	report := loadSample(t)
	hostile := "</script><script>alert(1)</script><img src=x onerror=alert(2)><!-- \u2028\u2029 ]]>"
	report.Title = hostile
	report.Resources[0].Label = hostile
	report.Resources[0].Address = hostile
	page := render(t, report)

	// Escaped text may still read "onerror=", but only real markup adds elements.
	for _, marker := range []string{"<script", "</script", "<img", "<!--"} {
		if strings.Count(page, marker) != strings.Count(baseline, marker) {
			t.Errorf("hostile text added %q markup to the document", marker)
		}
	}
	data := extractData(t, page)
	// Line terminators are harmless in HTML text but must never appear raw in script.
	if strings.ContainsAny(data, "\u2028\u2029") {
		t.Fatal("raw JavaScript line terminators reached the embedded data")
	}
	var decoded model.Report
	if err := json.Unmarshal([]byte(data), &decoded); err != nil {
		t.Fatalf("embedded data is not valid JSON: %v", err)
	}
	if decoded.Title != hostile || decoded.Resources[0].Label != hostile {
		t.Fatal("embedded data did not round-trip the original text")
	}
}

func TestStaticSummaryListsEveryChange(t *testing.T) {
	report := loadSample(t)
	page := render(t, report)
	for _, r := range report.Resources {
		if r.Change.Action == model.ActionNoOp {
			continue
		}
		if !strings.Contains(page, "<code>"+templateEscape(r.Address)+"</code>") {
			t.Errorf("static summary is missing %s", r.Address)
		}
	}
}

func TestNoticesAreInertAndPresent(t *testing.T) {
	page := render(t, loadSample(t))
	if !strings.Contains(page, `<template id="tfviz-notices"><pre>`) || !strings.Contains(page, "react-dom") {
		t.Fatal("third-party notices are missing")
	}
}

func TestRejectsAssetsThatWouldCloseTheirElement(t *testing.T) {
	assets := BundledAssets()
	assets.Script = append([]byte("var a='</SCRIPT>';"), assets.Script...)
	if err := Render(&bytes.Buffer{}, loadSample(t), assets); err == nil {
		t.Fatal("rendered a script containing </script")
	}
}

func extractData(t *testing.T, page string) string {
	t.Helper()
	const open = `<script type="application/json" id="tfviz-report">`
	start := strings.Index(page, open)
	if start < 0 {
		t.Fatal("report data element is missing")
	}
	rest := page[start+len(open):]
	return rest[:strings.Index(rest, "</script>")]
}

func templateEscape(s string) string {
	return strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;", `"`, "&#34;", `'`, "&#39;").Replace(s)
}
