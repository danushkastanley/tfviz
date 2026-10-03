// Command fixturegen produces real Terraform/OpenTofu producer JSON for the
// synthetic fixture stack without contacting AWS.
//
// It plans the "prior" variant against an empty state, then repeatedly
// "applies" synthetically: each created resource whose only unknown values
// are its own computed attributes receives deterministic fake values and is
// added to a version-4 state. Re-planning lets the producer propagate those
// values into dependents. Finally it plans the "proposed" variant against the
// synthetic state and exports plan and state JSON.
//
// Every producer command after init runs with a dead HTTPS proxy and fake
// credentials, so any attempted AWS call fails instead of leaving the machine.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const maxIterations = 25

func main() {
	tool := flag.String("tool", "terraform", "producer binary: terraform or tofu")
	stack := flag.String("stack", "testdata/stacks/aws-review", "fixture stack directory")
	out := flag.String("out", "", "output directory for producer JSON")
	cache := flag.String("cache", ".cache", "working directory for provider caches")
	flag.Parse()

	if *out == "" {
		fmt.Fprintln(os.Stderr, "fixturegen: -out is required")
		os.Exit(2)
	}
	if err := run(*tool, *stack, *out, *cache); err != nil {
		fmt.Fprintln(os.Stderr, "fixturegen:", err)
		os.Exit(1)
	}
}

func run(tool, stack, out, cache string) error {
	p, err := newProducer(tool, stack, cache)
	if err != nil {
		return err
	}
	if err := p.init(); err != nil {
		return err
	}
	version, err := p.version()
	if err != nil {
		return err
	}
	var schemas schemaJSON
	if err := p.jsonOutput(&schemas, "providers", "schema", "-json"); err != nil {
		return err
	}

	statePath := filepath.Join(p.work, "prior.tfstate")
	state := newState(version)
	for iteration := 1; ; iteration++ {
		if iteration > maxIterations {
			return errors.New("synthetic apply did not converge")
		}
		if err := writeJSON(statePath, state); err != nil {
			return err
		}
		plan, err := p.plan("prior", statePath)
		if err != nil {
			return err
		}
		added, pending, err := synthesiseReady(plan, schemas, state)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "iteration %d: synthesised %d, pending %d\n", iteration, added, pending)
		if pending == 0 && added == 0 {
			if err := requireNoOps(plan); err != nil {
				return err
			}
			break
		}
		if added == 0 {
			return fmt.Errorf("%d resources can never become ready", pending)
		}
		state.sortDeterministically()
	}

	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	// Store the producer's own serialisation of the state, so the raw-state
	// fixture is authentic producer output rather than ours.
	if err := p.pullState(statePath, filepath.Join(out, "prior.tfstate")); err != nil {
		return err
	}
	if err := p.planToFile("proposed", statePath, filepath.Join(out, "plan.json")); err != nil {
		return err
	}
	if err := p.showToFile(statePath, filepath.Join(out, "state.json")); err != nil {
		return err
	}
	return writeVersions(filepath.Join(out, "VERSIONS"), tool, version, p.providerVersion())
}

// synthesiseReady adds every create-only resource whose unknowns are all
// self-computed. It returns how many were added and how many still wait.
func synthesiseReady(plan *planJSON, schemas schemaJSON, state *stateFile) (added, pending int, err error) {
	for _, rc := range plan.ResourceChanges {
		if rc.Mode != "managed" || !slices.Equal(rc.Change.Actions, []string{"create"}) {
			continue
		}
		var leaves [][]pathStep
		unknownLeaves(rc.Change.AfterUnknown, nil, &leaves)
		expressions := plan.configExpressions(rc)
		ready := true
		for _, leaf := range leaves {
			if derivedFromConfig(expressions, leaf) {
				ready = false
				break
			}
		}
		if !ready {
			pending++
			continue
		}
		schema, ok := schemas.ProviderSchemas[rc.ProviderName].ResourceSchemas[rc.Type]
		if !ok {
			return 0, 0, fmt.Errorf("no schema for %s", rc.Type)
		}
		attrs, err := fillUnknowns(rc, schema.Block, leaves)
		if err != nil {
			return 0, 0, err
		}
		state.add(rc, schema.Version, attrs)
		added++
	}
	return added, pending, nil
}

// requireNoOps confirms the synthetic state is a faithful "applied" snapshot:
// planning the same configuration again must change nothing.
func requireNoOps(plan *planJSON) error {
	var drift []string
	for _, rc := range plan.ResourceChanges {
		if rc.Mode == "managed" && !slices.Equal(rc.Change.Actions, []string{"no-op"}) {
			drift = append(drift, fmt.Sprintf("%s %v", rc.Address, rc.Change.Actions))
		}
	}
	if len(drift) > 0 {
		return fmt.Errorf("synthetic state is not stable: %v", drift)
	}
	return nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writeVersions(path, tool, producerVersion, providerVersion string) error {
	content := fmt.Sprintf("producer=%s\nproducer_version=%s\nprovider=hashicorp/aws\nprovider_version=%s\n",
		tool, producerVersion, providerVersion)
	return os.WriteFile(path, []byte(content), 0o644)
}
