package input

import (
	"encoding/json"
	"strconv"
	"strings"
)

// rawStateDocument is a version-4 state snapshot (terraform.tfstate). Root
// outputs are deliberately not decoded: they are never shown.
type rawStateDocument struct {
	Version          int              `json:"version"`
	TerraformVersion string           `json:"terraform_version"`
	Resources        []rawStateEntity `json:"resources"`
}

type rawStateEntity struct {
	Module    string             `json:"module"`
	Mode      string             `json:"mode"`
	Type      string             `json:"type"`
	Name      string             `json:"name"`
	Provider  string             `json:"provider"`
	Instances []rawStateInstance `json:"instances"`
}

type rawStateInstance struct {
	IndexKey            any               `json:"index_key"`
	Deposed             string            `json:"deposed"`
	Attributes          any               `json:"attributes"`
	SensitiveAttributes []json.RawMessage `json:"sensitive_attributes"`
}

func readRawState(data []byte) (*Snapshot, error) {
	var doc rawStateDocument
	if err := decode(data, &doc); err != nil {
		return nil, err
	}
	if doc.Version != 4 {
		return nil, newError(CodeUnsupported, "This state file uses format version "+strconv.Itoa(min(max(doc.Version, 0), 99))+
			", and tfviz reads version 4 directly. Export it with `terraform show -json` or `tofu show -json` instead.")
	}
	snap := &Snapshot{
		Kind:            KindState,
		Producer:        ProducerUnknown,
		FormatVersion:   "raw-v4",
		ProducerVersion: safeVersion(doc.TerraformVersion),
	}
	for _, entity := range doc.Resources {
		source := strings.TrimSuffix(strings.TrimPrefix(providerAddress(entity.Provider), `provider["`), `"]`)
		for _, inst := range entity.Instances {
			address := rawAddress(entity, inst.IndexKey)
			if inst.Deposed != "" {
				snap.Notices = append(snap.Notices, Notice{Code: NoticeDeposedSkipped, Address: address,
					Message: "A deposed object from an earlier interrupted replacement is not shown."})
				continue
			}
			if len(snap.Resources) >= MaxResources {
				return nil, newError(CodeTooManyThings, "The state has more resources than tfviz can process in one report (50,000).")
			}
			snap.Resources = append(snap.Resources, Resource{
				Address:  address,
				Module:   entity.Module,
				Mode:     entity.Mode,
				Type:     entity.Type,
				Name:     entity.Name,
				Provider: providerSource(source),
				After:    buildValue(inst.Attributes, sensitivityMarks(inst.SensitiveAttributes), nil, nil, nil),
				HasAfter: true,
			})
			if snap.Producer == ProducerUnknown {
				snap.Producer = producerFromName(source)
			}
		}
	}
	sortResources(snap.Resources)
	return snap, nil
}

// providerAddress drops a module prefix and an alias suffix:
// `module.a.provider["registry.terraform.io/hashicorp/aws"].west` → `provider["…"]`.
func providerAddress(p string) string {
	start := strings.Index(p, `provider["`)
	if start < 0 {
		return ""
	}
	p = p[start:]
	if end := strings.Index(p, `"]`); end >= 0 {
		return p[:end+2]
	}
	return p
}

func rawAddress(e rawStateEntity, key any) string {
	local := e.Type + "." + e.Name
	if e.Mode == "data" {
		local = "data." + local
	}
	switch k := key.(type) {
	case string:
		quoted, _ := json.Marshal(k)
		local += "[" + string(quoted) + "]"
	case json.Number:
		local += "[" + k.String() + "]"
	}
	if e.Module != "" {
		return e.Module + "." + local
	}
	return local
}

// pathStep is one step of a state attribute path.
type pathStep struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

// sensitivityMarks converts state sensitive_attributes paths into the marker
// tree buildValue expects. Unrecognised path steps mark the whole value, so
// a format change can never expose a secret.
func sensitivityMarks(paths []json.RawMessage) any {
	var root any
	for _, raw := range paths {
		var steps []pathStep
		if err := json.Unmarshal(raw, &steps); err != nil || len(steps) == 0 {
			return true
		}
		marks, ok := markPath(root, steps)
		if !ok {
			return true
		}
		root = marks
	}
	return root
}

func markPath(node any, steps []pathStep) (any, bool) {
	if len(steps) == 0 || marked(node) {
		return true, true
	}
	step := steps[0]
	switch step.Type {
	case "get_attr":
		var name string
		if json.Unmarshal(step.Value, &name) != nil {
			return nil, false
		}
		return markKey(node, name, steps[1:])
	case "index":
		var key struct {
			Value json.RawMessage `json:"value"`
			Type  string          `json:"type"`
		}
		if json.Unmarshal(step.Value, &key) != nil {
			return nil, false
		}
		if key.Type == "number" {
			var i int
			if json.Unmarshal(key.Value, &i) != nil || i < 0 || i > 100_000 {
				return nil, false
			}
			list, _ := node.([]any)
			for len(list) <= i {
				list = append(list, nil)
			}
			child, ok := markPath(list[i], steps[1:])
			list[i] = child
			return list, ok
		}
		var name string
		if json.Unmarshal(key.Value, &name) != nil {
			return nil, false
		}
		return markKey(node, name, steps[1:])
	}
	return nil, false
}

func markKey(node any, name string, rest []pathStep) (any, bool) {
	m, _ := node.(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	child, ok := markPath(m[name], rest)
	m[name] = child
	return m, ok
}
