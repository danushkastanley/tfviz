package main

import "encoding/json"

// The minimal subset of producer JSON that fixture generation needs. This is
// development tooling only; the product reader lives in internal/input.

type planJSON struct {
	ResourceChanges []resourceChange `json:"resource_changes"`
	Configuration   struct {
		RootModule configModule `json:"root_module"`
	} `json:"configuration"`
}

type resourceChange struct {
	Address       string          `json:"address"`
	ModuleAddress string          `json:"module_address"`
	Mode          string          `json:"mode"`
	Type          string          `json:"type"`
	Name          string          `json:"name"`
	Index         json.RawMessage `json:"index"`
	ProviderName  string          `json:"provider_name"`
	Change        struct {
		Actions        []string `json:"actions"`
		After          any      `json:"after"`
		AfterUnknown   any      `json:"after_unknown"`
		AfterSensitive any      `json:"after_sensitive"`
	} `json:"change"`
}

type configModule struct {
	Resources   []configResource      `json:"resources"`
	ModuleCalls map[string]moduleCall `json:"module_calls"`
}

type moduleCall struct {
	Module configModule `json:"module"`
}

type configResource struct {
	Mode        string         `json:"mode"`
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Expressions map[string]any `json:"expressions"`
}

type schemaJSON struct {
	ProviderSchemas map[string]struct {
		ResourceSchemas map[string]resourceSchema `json:"resource_schemas"`
	} `json:"provider_schemas"`
}

type resourceSchema struct {
	Version int         `json:"version"`
	Block   schemaBlock `json:"block"`
}

type schemaBlock struct {
	Attributes map[string]schemaAttribute `json:"attributes"`
	BlockTypes map[string]struct {
		Block schemaBlock `json:"block"`
	} `json:"block_types"`
}

type schemaAttribute struct {
	Type json.RawMessage `json:"type"`
}

// configExpressions finds the configuration expressions for a resource
// change, descending into module calls by the change's module address.
func (p *planJSON) configExpressions(rc resourceChange) map[string]any {
	mod := p.Configuration.RootModule
	for _, name := range moduleNames(rc.ModuleAddress) {
		call, ok := mod.ModuleCalls[name]
		if !ok {
			return nil
		}
		mod = call.Module
	}
	for _, r := range mod.Resources {
		if r.Mode == rc.Mode && r.Type == rc.Type && r.Name == rc.Name {
			return r.Expressions
		}
	}
	return nil
}
