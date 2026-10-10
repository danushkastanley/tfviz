package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// page is one rendered page. Path is the public URL; pages are written as
// <path>/index.html so every host serves them without rewrites.
type page struct {
	Path        string
	Template    string
	Title       string
	Description string
	Nav         string // nav label; empty keeps the page out of the header
}

var pages = []page{
	{"/", "index", "tfviz: see your Terraform plan before you apply it",
		"Turn Terraform and OpenTofu plans and state into a calm, interactive architecture diagram. One offline HTML file, with secrets left out by design.", ""},
	{"/features/", "features", "Features · tfviz",
		"Plan reviews, state reports, a private local explorer, safe sharing, module views and official AWS icons.", "Features"},
	{"/demo/", "demo", "Live demo · tfviz",
		"Open real tfviz reports generated from synthetic Terraform and OpenTofu stacks.", "Demo"},
	{"/get-started/", "get-started", "Get started · tfviz",
		"Install tfviz, verify the download and generate your first report in a few minutes.", ""},
	{"/ci/", "ci", "Use tfviz in CI · tfviz",
		"Generate a review report in GitHub Actions, GitLab CI, Jenkins or Buildkite, and upload only the report.", "CI"},
	{"/security/", "security", "Security and privacy · tfviz",
		"How tfviz keeps secrets out of reports, what it never does, and where the limits are.", "Security"},
	{"/faq/", "faq", "Questions · tfviz",
		"Answers to common questions about tfviz, privacy, supported providers and sharing reports.", "FAQ"},
	{"/404.html", "404", "Page not found · tfviz", "This page does not exist.", ""},
}

// pageData is what every template sees.
type pageData struct {
	Page   page
	Pages  []page
	Config config
	CSS    string
	CSP    string
}

func renderPages(files map[string][]byte, cfg config, css string) error {
	layout := filepath.Join(root, "templates", "layout.html")
	for _, p := range pages {
		t, err := template.New("layout.html").Funcs(template.FuncMap{
			"abs": func(path string) string { return strings.TrimSuffix(cfg.BaseURL, "/") + path },
		}).ParseFiles(layout, filepath.Join(root, "templates", "icons.html"), filepath.Join(root, "templates", p.Template+".html"))
		if err != nil {
			return err
		}
		var out bytes.Buffer
		data := pageData{Page: p, Pages: pages, Config: cfg, CSS: css, CSP: metaCSP}
		if err := t.Execute(&out, data); err != nil {
			return fmt.Errorf("%s: %w", p.Template, err)
		}
		files[outputPath(p.Path)] = out.Bytes()
	}
	return nil
}

func outputPath(urlPath string) string {
	if strings.HasSuffix(urlPath, ".html") {
		return "public" + urlPath
	}
	return "public" + urlPath + "index.html"
}

// fingerprint publishes a static file under a content-hashed name, so it
// can be cached indefinitely, and returns its URL.
func fingerprint(files map[string][]byte, src, dst string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, src))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	name := dst + "." + hex.EncodeToString(sum[:])[:10] + filepath.Ext(src)
	files["public/"+name] = data
	return "/" + name, nil
}

func robots(cfg config) []byte {
	out := "User-agent: *\nAllow: /\n"
	if cfg.BaseURL != "" {
		out += "Sitemap: " + strings.TrimSuffix(cfg.BaseURL, "/") + "/sitemap.xml\n"
	}
	return []byte(out)
}

func sitemap(cfg config) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, p := range pages {
		if strings.HasSuffix(p.Path, ".html") {
			continue
		}
		fmt.Fprintf(&b, "  <url><loc>%s%s</loc></url>\n", strings.TrimSuffix(cfg.BaseURL, "/"), p.Path)
	}
	b.WriteString("</urlset>\n")
	return []byte(b.String())
}
