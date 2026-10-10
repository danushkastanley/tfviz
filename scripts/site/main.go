// Command site renders the tfviz website from site/ into site/public, plus
// the hosting configuration for Vercel (site/vercel.json) and Cloudflare
// (site/public/_headers). The output is committed, so hosts serve it as is
// with no build step.
//
//	go run ./scripts/site          write the site
//	go run ./scripts/site -check   fail if the committed output is stale
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const root = "site"

// config is site/site.json. BaseURL is empty until the domain is chosen;
// absolute links (canonical, Open Graph, sitemap) are written only once it
// is set.
type config struct {
	BaseURL string `json:"base_url"`
	Version string `json:"version"`
	Repo    string `json:"repo"`
}

func main() {
	check := flag.Bool("check", false, "fail if site/public or site/vercel.json are out of date")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, "site:", err)
		os.Exit(1)
	}
}

func run(check bool) error {
	var cfg config
	raw, err := os.ReadFile(filepath.Join(root, "site.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("site.json: %w", err)
	}
	files, err := build(cfg)
	if err != nil {
		return err
	}
	if check {
		return compare(files)
	}
	return write(files)
}

// build returns every output file, keyed by its path under site/.
func build(cfg config) (map[string][]byte, error) {
	files := map[string][]byte{}
	if err := copyStatic(files); err != nil {
		return nil, err
	}
	css, err := fingerprint(files, "static/site.css", "assets/site")
	if err != nil {
		return nil, err
	}
	if err := renderPages(files, cfg, css); err != nil {
		return nil, err
	}
	files["public/_headers"] = cloudflareHeaders()
	vercel, err := vercelConfig()
	if err != nil {
		return nil, err
	}
	files["vercel.json"] = vercel
	files["public/robots.txt"] = robots(cfg)
	if cfg.BaseURL != "" {
		files["public/sitemap.xml"] = sitemap(cfg)
	}
	return files, nil
}

// copyStatic copies site/static into public, except the stylesheet, which
// is published under a content hash.
func copyStatic(files map[string][]byte) error {
	dir := filepath.Join(root, "static")
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == ".DS_Store" {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "site.css" {
			return nil
		}
		data, err := os.ReadFile(path)
		files[filepath.ToSlash(filepath.Join("public", rel))] = data
		return err
	})
}

// write replaces site/public and site/vercel.json with the built files.
func write(files map[string][]byte) error {
	if err := os.RemoveAll(filepath.Join(root, "public")); err != nil {
		return err
	}
	for _, name := range sortedKeys(files) {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, files[name], 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("Wrote %d files to %s/public\n", len(files), root)
	return nil
}

// compare reports any difference between the built files and the committed
// ones, including files that should no longer exist.
func compare(files map[string][]byte) error {
	var stale []string
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil || !bytes.Equal(got, want) {
			stale = append(stale, name)
		}
	}
	_ = filepath.WalkDir(filepath.Join(root, "public"), func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(root, path)
			if _, ok := files[filepath.ToSlash(rel)]; !ok {
				stale = append(stale, filepath.ToSlash(rel)+" (should not exist)")
			}
		}
		return nil
	})
	if len(stale) > 0 {
		sort.Strings(stale)
		return fmt.Errorf("out of date; run `make site`:\n  %s", joinLines(stale))
	}
	return nil
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func joinLines(lines []string) string {
	var b bytes.Buffer
	for i, l := range lines {
		if i > 0 {
			b.WriteString("\n  ")
		}
		b.WriteString(l)
	}
	return b.String()
}
