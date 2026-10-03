package main

import (
	"encoding/json"
	"sort"
	"strings"
)

// Version-4 state structures, written by fixture generation only.

type stateFile struct {
	Version          int             `json:"version"`
	TerraformVersion string          `json:"terraform_version"`
	Serial           int             `json:"serial"`
	Lineage          string          `json:"lineage"`
	Outputs          map[string]any  `json:"outputs"`
	Resources        []stateResource `json:"resources"`
	CheckResults     any             `json:"check_results"`
}

type stateResource struct {
	Module    string          `json:"module,omitempty"`
	Mode      string          `json:"mode"`
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Provider  string          `json:"provider"`
	Instances []stateInstance `json:"instances"`
}

type stateInstance struct {
	IndexKey            json.RawMessage `json:"index_key,omitempty"`
	SchemaVersion       int             `json:"schema_version"`
	Attributes          map[string]any  `json:"attributes"`
	SensitiveAttributes []any           `json:"sensitive_attributes"`
}

// A fixed lineage keeps regenerated fixtures byte-stable.
const fixtureLineage = "00000000-0000-4000-8000-00000000f1c5"

func newState(producerVersion string) *stateFile {
	return &stateFile{
		Version:          4,
		TerraformVersion: producerVersion,
		Serial:           1,
		Lineage:          fixtureLineage,
		Outputs:          map[string]any{},
		Resources:        []stateResource{},
	}
}

// add records one synthesised resource instance, grouping instances of the
// same resource block together as Terraform does.
func (s *stateFile) add(rc resourceChange, schemaVersion int, attrs map[string]any) {
	instance := stateInstance{SchemaVersion: schemaVersion, Attributes: attrs, SensitiveAttributes: []any{}}
	sensitivePaths(rc.Change.AfterSensitive, nil, &instance.SensitiveAttributes)
	if len(rc.Index) > 0 {
		instance.IndexKey = rc.Index
	}
	for i := range s.Resources {
		r := &s.Resources[i]
		if r.Module == rc.ModuleAddress && r.Mode == rc.Mode && r.Type == rc.Type && r.Name == rc.Name {
			r.Instances = append(r.Instances, instance)
			return
		}
	}
	s.Resources = append(s.Resources, stateResource{
		Module:    rc.ModuleAddress,
		Mode:      rc.Mode,
		Type:      rc.Type,
		Name:      rc.Name,
		Provider:  `provider["` + rc.ProviderName + `"]`,
		Instances: []stateInstance{instance},
	})
}

// sortDeterministically orders resources and instances so output is stable
// regardless of the iteration in which each was synthesised.
func (s *stateFile) sortDeterministically() {
	key := func(r stateResource) string {
		return strings.Join([]string{r.Module, r.Mode, r.Type, r.Name}, "\x00")
	}
	sort.SliceStable(s.Resources, func(i, j int) bool { return key(s.Resources[i]) < key(s.Resources[j]) })
	for i := range s.Resources {
		inst := s.Resources[i].Instances
		sort.SliceStable(inst, func(a, b int) bool { return string(inst[a].IndexKey) < string(inst[b].IndexKey) })
	}
}

// sensitivePaths converts after_sensitive markers into the state's
// sensitive_attributes path encoding, so re-planning sees identical marks.
func sensitivePaths(marks any, prefix []any, out *[]any) {
	switch m := marks.(type) {
	case bool:
		if m && len(prefix) > 0 {
			*out = append(*out, append([]any(nil), prefix...))
		}
	case map[string]any:
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			step := map[string]any{"type": "get_attr", "value": k}
			sensitivePaths(m[k], append(prefix, step), out)
		}
	case []any:
		for i, v := range m {
			step := map[string]any{"type": "index", "value": map[string]any{"value": i, "type": "number"}}
			sensitivePaths(v, append(prefix, step), out)
		}
	}
}

// moduleNames converts "module.a[0].module.b" into ["a", "b"].
func moduleNames(address string) []string {
	if address == "" {
		return nil
	}
	var names []string
	parts := strings.Split(address, ".")
	for i := 0; i+1 < len(parts); i += 2 {
		name := parts[i+1]
		if cut := strings.IndexByte(name, '['); cut >= 0 {
			name = name[:cut]
		}
		names = append(names, name)
	}
	return names
}
