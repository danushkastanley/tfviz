package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Synthetic canary secrets supplied as sensitive variables. They are listed in
// testdata/canaries.txt so tests can prove they never reach a report.
var canaryVariables = []string{
	"TF_VAR_db_password=CANARY-DB-PASSWORD-a41c",
	"TF_VAR_kafka_password=CANARY-KAFKA-PASSWORD-e07b",
}

// deadProxy routes all HTTP(S) to a closed local port. It applies only to the
// producer processes started here, never to the user's shell or system.
var deadProxy = []string{
	"HTTPS_PROXY=http://127.0.0.1:9",
	"HTTP_PROXY=http://127.0.0.1:9",
	"NO_PROXY=",
}

type producer struct {
	tool  string
	stack string
	work  string
	data  string
	cache string
}

func newProducer(tool, stack, cache string) (*producer, error) {
	if tool != "terraform" && tool != "tofu" {
		return nil, fmt.Errorf("unsupported tool %q", tool)
	}
	root, err := filepath.Abs(filepath.Join(cache, "fixtures", tool))
	if err != nil {
		return nil, err
	}
	p := &producer{
		tool:  tool,
		stack: filepath.Join(root, "stack"),
		work:  filepath.Join(root, "work"),
		data:  filepath.Join(root, "data"),
		cache: filepath.Join(root, "plugins"),
	}
	for _, dir := range []string{p.work, p.data, p.cache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	// Each producer plans a private copy so lock files and working state never
	// mix between Terraform and OpenTofu or land in testdata.
	if err := os.RemoveAll(p.stack); err != nil {
		return nil, err
	}
	return p, os.CopyFS(p.stack, os.DirFS(stack))
}

// baseEnv drops ambient AWS and proxy settings so the user's real credentials
// and profiles can never be picked up by the fixture stack.
func (p *producer) baseEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		name := strings.ToUpper(kv[:strings.IndexByte(kv, '=')])
		if strings.HasPrefix(name, "AWS_") || strings.HasPrefix(name, "TF_") || strings.HasSuffix(name, "_PROXY") {
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		"TF_DATA_DIR="+p.data,
		"TF_PLUGIN_CACHE_DIR="+p.cache,
		"TF_IN_AUTOMATION=1",
		"CHECKPOINT_DISABLE=1",
		// Obviously fake credentials that do not resemble real key formats.
		"AWS_ACCESS_KEY_ID=fixture-access-key-id",
		"AWS_SECRET_ACCESS_KEY=fixture-secret-access-key",
	)
}

func (p *producer) command(offline bool, args ...string) *exec.Cmd {
	cmd := exec.Command(p.tool, args...)
	cmd.Dir = p.stack
	cmd.Env = append(p.baseEnv(), canaryVariables...)
	if offline {
		cmd.Env = append(cmd.Env, deadProxy...)
	}
	return cmd
}

func (p *producer) runQuiet(offline bool, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := p.command(offline, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %s: %w\n%s", p.tool, strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// init is the only step allowed network access: it downloads the pinned provider.
func (p *producer) init() error {
	_, err := p.runQuiet(false, "init", "-input=false", "-no-color")
	return err
}

func (p *producer) jsonOutput(v any, args ...string) error {
	out, err := p.runQuiet(true, args...)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, v)
}

func (p *producer) version() (string, error) {
	var v struct {
		Version string `json:"terraform_version"`
	}
	if err := p.jsonOutput(&v, "version", "-json"); err != nil {
		return "", err
	}
	return v.Version, nil
}

func (p *producer) providerVersion() string {
	var v struct {
		Selections map[string]string `json:"provider_selections"`
	}
	if err := p.jsonOutput(&v, "version", "-json"); err != nil {
		return "unknown"
	}
	for addr, version := range v.Selections {
		if strings.HasSuffix(addr, "/hashicorp/aws") {
			return version
		}
	}
	return "unknown"
}

func (p *producer) savePlan(phase, statePath string) (string, error) {
	planPath := filepath.Join(p.work, phase+".tfplan")
	_, err := p.runQuiet(true, "plan", "-input=false", "-no-color", "-refresh=false",
		"-lock=false", "-var", "phase="+phase, "-state="+statePath, "-out="+planPath)
	return planPath, err
}

func (p *producer) plan(phase, statePath string) (*planJSON, error) {
	planPath, err := p.savePlan(phase, statePath)
	if err != nil {
		return nil, err
	}
	var plan planJSON
	return &plan, p.jsonOutput(&plan, "show", "-json", planPath)
}

func (p *producer) planToFile(phase, statePath, dest string) error {
	planPath, err := p.savePlan(phase, statePath)
	if err != nil {
		return err
	}
	return p.showToFile(planPath, dest)
}

func (p *producer) showToFile(source, dest string) error {
	out, err := p.runQuiet(true, "show", "-json", source)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, out, 0o644)
}
