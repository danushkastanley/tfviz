package input

import (
	"sort"
	"strings"
)

type configurationJSON struct {
	ProviderConfig map[string]struct {
		Expressions map[string]any `json:"expressions"`
	} `json:"provider_config"`
	RootModule moduleConfigJSON `json:"root_module"`
}

type moduleConfigJSON struct {
	Resources []struct {
		Address           string         `json:"address"`
		Mode              string         `json:"mode"`
		Type              string         `json:"type"`
		Name              string         `json:"name"`
		ProviderConfigKey string         `json:"provider_config_key"`
		Expressions       map[string]any `json:"expressions"`
	} `json:"resources"`
	ModuleCalls map[string]struct {
		Expressions map[string]any   `json:"expressions"`
		Module      moduleConfigJSON `json:"module"`
	} `json:"module_calls"`
}

// Config indexes configuration references. Only references are kept, never
// constant values: configuration literals can contain secrets.
type Config struct {
	resources map[string]configResource
	// callArgs holds a module call's argument references, keyed by the
	// module's static path ("module.a.module.b") and argument name.
	callArgs map[string]map[string][]string
}

type configResource struct {
	providerKey string
	refs        map[string][]string // dotted attribute path → references
}

func buildConfig(doc *configurationJSON) *Config {
	c := &Config{resources: map[string]configResource{}, callArgs: map[string]map[string][]string{}}
	c.index("", doc.RootModule)
	return c
}

func (c *Config) index(path string, m moduleConfigJSON) {
	for _, r := range m.Resources {
		refs := map[string][]string{}
		flattenRefs("", r.Expressions, refs)
		c.resources[configKey(path, r.Mode, r.Type, r.Name)] = configResource{providerKey: r.ProviderConfigKey, refs: refs}
	}
	for name, call := range m.ModuleCalls {
		child := joinPath(path, "module."+name)
		args := map[string][]string{}
		flattenRefs("", call.Expressions, args)
		c.callArgs[child] = args
		c.index(child, call.Module)
	}
}

// flattenRefs records the references of every expression under prefix.
// Nested blocks are arrays of expression objects; their index is dropped
// because configuration does not know how many instances exist.
func flattenRefs(prefix string, node any, out map[string][]string) {
	switch n := node.(type) {
	case map[string]any:
		if refs, ok := n["references"].([]any); ok {
			for _, ref := range refs {
				if s, ok := ref.(string); ok {
					out[prefix] = append(out[prefix], s)
				}
			}
			return
		}
		if _, isConstant := n["constant_value"]; isConstant {
			return
		}
		for key, child := range n {
			flattenRefs(joinField(prefix, key), child, out)
		}
	case []any:
		for _, child := range n {
			flattenRefs(prefix, child, out)
		}
	}
}

// References returns the absolute addresses an attribute's configuration
// refers to, following module variables into the calling module. Locals,
// each/count values and other indirections are not resolved, so callers
// must treat a missing reference as unresolved, never as absent.
func (c *Config) References(r Resource, attr string) []string {
	if c == nil {
		return nil
	}
	res, ok := c.resources[configKey(staticPath(r.Module), r.Mode, r.Type, r.Name)]
	if !ok {
		return nil
	}
	// Attributes written as blocks (such as `route` or `ingress`) record
	// their references on the whole attribute, so fall back to the nearest
	// ancestor expression. Callers filter matches by resource type.
	refs, ok := res.refs[attr]
	for path := attr; !ok && strings.Contains(path, "."); {
		path = path[:strings.LastIndex(path, ".")]
		refs, ok = res.refs[path]
	}
	seen := map[string]bool{}
	var out []string
	c.resolve(r.Module, refs, seen, &out, 0)
	sort.Strings(out)
	return out
}

func (c *Config) resolve(moduleInstance string, refs []string, seen map[string]bool, out *[]string, depth int) {
	if depth > 16 {
		return
	}
	for _, ref := range refs {
		switch {
		case strings.HasPrefix(ref, "var."):
			name, _, _ := strings.Cut(strings.TrimPrefix(ref, "var."), ".")
			name, _, _ = strings.Cut(name, "[")
			if moduleInstance == "" {
				continue // root variables are inputs, not resources
			}
			args := c.callArgs[staticPath(moduleInstance)]
			c.resolve(parentInstance(moduleInstance), args[name], seen, out, depth+1)
		case strings.HasPrefix(ref, "local."), strings.HasPrefix(ref, "each."), strings.HasPrefix(ref, "count."),
			strings.HasPrefix(ref, "path."), strings.HasPrefix(ref, "terraform."), strings.HasPrefix(ref, "module."):
			continue
		default:
			abs := joinPath(moduleInstance, ref)
			if !seen[abs] {
				seen[abs] = true
				*out = append(*out, abs)
			}
		}
	}
}

func (doc *configurationJSON) constantRegions() map[string]string {
	regions := map[string]string{}
	for key, p := range doc.ProviderConfig {
		if expr, ok := p.Expressions["region"].(map[string]any); ok {
			if region, ok := expr["constant_value"].(string); ok && region != "" {
				regions[key] = region
			}
		}
	}
	return regions
}

func assignProviderKeys(snap *Snapshot) {
	if snap.Config == nil {
		return
	}
	for i := range snap.Resources {
		r := &snap.Resources[i]
		if res, ok := snap.Config.resources[configKey(staticPath(r.Module), r.Mode, r.Type, r.Name)]; ok {
			r.ProviderKey = res.providerKey
		}
	}
}

func configKey(path, mode, typ, name string) string {
	return path + "|" + mode + "." + typ + "." + name
}

// staticPath removes instance keys: `module.a["x"].module.b[0]` → `module.a.module.b`.
func staticPath(instance string) string {
	var b strings.Builder
	depth := 0
	for _, r := range instance {
		switch {
		case r == '[':
			depth++
		case r == ']':
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parentInstance drops the last `module.name[key]` segment.
func parentInstance(instance string) string {
	i := strings.LastIndex(instance, "module.")
	if i <= 0 {
		return ""
	}
	return strings.TrimSuffix(instance[:i], ".")
}

func joinPath(prefix, rest string) string {
	if prefix == "" {
		return rest
	}
	return prefix + "." + rest
}

func joinField(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}
