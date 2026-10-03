package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// pathStep is one step into a planned value: an object key or a list index.
type pathStep struct {
	key   string
	index int
	isKey bool
}

func (s pathStep) String() string {
	if s.isKey {
		return s.key
	}
	return fmt.Sprintf("[%d]", s.index)
}

// unknownLeaves walks after_unknown and returns every path marked unknown.
func unknownLeaves(marks any, prefix []pathStep, out *[][]pathStep) {
	switch m := marks.(type) {
	case bool:
		if m {
			*out = append(*out, append([]pathStep(nil), prefix...))
		}
	case map[string]any:
		for k, v := range m {
			unknownLeaves(v, append(prefix, pathStep{key: k, isKey: true}), out)
		}
	case []any:
		for i, v := range m {
			unknownLeaves(v, append(prefix, pathStep{index: i}), out)
		}
	}
}

// derivedFromConfig reports whether a configuration expression covers the
// path. Such values are unknown only because they reference something not yet
// in the synthetic state, so Terraform resolves them on a later iteration.
func derivedFromConfig(expressions map[string]any, path []pathStep) bool {
	var node any = expressions
	for _, step := range path {
		if expr, ok := node.(map[string]any); ok {
			if _, isExpr := expr["references"]; isExpr {
				return true
			}
		}
		switch n := node.(type) {
		case map[string]any:
			if !step.isKey {
				return false
			}
			node = n[step.key]
		case []any:
			if step.isKey || step.index >= len(n) {
				return false
			}
			node = n[step.index]
		default:
			return false
		}
	}
	expr, ok := node.(map[string]any)
	if !ok {
		return false
	}
	_, isExpr := expr["references"]
	return isExpr
}

// fillUnknowns replaces each unknown leaf in after with a deterministic
// synthetic value appropriate to the attribute name and schema type.
func fillUnknowns(rc resourceChange, schema schemaBlock, leaves [][]pathStep) (map[string]any, error) {
	after, ok := rc.Change.After.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: planned value is not an object", rc.Address)
	}
	// Identity attributes first, because other values (such as ARNs) embed them.
	ordered := orderIdentityFirst(leaves)
	for _, path := range ordered {
		value := synthesise(rc, after, schema, path)
		if err := setPath(after, path, value); err != nil {
			return nil, fmt.Errorf("%s: %w", rc.Address, err)
		}
	}
	return after, nil
}

func orderIdentityFirst(leaves [][]pathStep) [][]pathStep {
	rank := func(p []pathStep) int {
		if len(p) == 1 && p[0].key == "id" {
			return 0
		}
		if len(p) == 1 && p[0].key == "arn" {
			return 1
		}
		return 2
	}
	out := make([][]pathStep, 0, len(leaves))
	for r := 0; r <= 2; r++ {
		for _, p := range leaves {
			if rank(p) == r {
				out = append(out, p)
			}
		}
	}
	return out
}

func setPath(root map[string]any, path []pathStep, value any) error {
	var parent any = root
	for i, step := range path {
		last := i == len(path)-1
		switch p := parent.(type) {
		case map[string]any:
			if last {
				p[step.key] = value
				return nil
			}
			parent = p[step.key]
		case []any:
			if step.index >= len(p) {
				return fmt.Errorf("index %d out of range", step.index)
			}
			if last {
				p[step.index] = value
				return nil
			}
			parent = p[step.index]
		default:
			return fmt.Errorf("cannot descend into %T", parent)
		}
	}
	return nil
}

// attributeType resolves the cty type JSON of the attribute at path.
func attributeType(block schemaBlock, path []pathStep) json.RawMessage {
	for i, step := range path {
		if !step.isKey {
			continue
		}
		if attr, ok := block.Attributes[step.key]; ok {
			if i == len(path)-1 {
				return attr.Type
			}
			return nil
		}
		nested, ok := block.BlockTypes[step.key]
		if !ok {
			return nil
		}
		if i == len(path)-1 {
			// A whole nested block is unknown: an empty block list is its zero value.
			return json.RawMessage(`["list","dynamic"]`)
		}
		block = nested.Block
	}
	return nil
}

func zeroForType(t json.RawMessage) any {
	s := strings.TrimSpace(string(t))
	switch {
	case s == `"number"`:
		return 0
	case s == `"bool"`:
		return false
	case strings.HasPrefix(s, `["list"`), strings.HasPrefix(s, `["set"`):
		return []any{}
	case strings.HasPrefix(s, `["map"`):
		return map[string]any{}
	case strings.HasPrefix(s, `["object"`):
		return nil
	default:
		return ""
	}
}
