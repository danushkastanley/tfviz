package graph

import (
	"sort"
	"strings"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// resolver matches recorded references to resources.
type resolver struct {
	snap      *input.Snapshot
	byValue   map[string][]*Node // identity attribute value → nodes
	byAddress map[string]*Node
}

func newResolver(snap *input.Snapshot, nodes []*Node) *resolver {
	r := &resolver{snap: snap, byValue: map[string][]*Node{}, byAddress: map[string]*Node{}}
	for _, n := range nodes {
		r.byAddress[n.Resource.Address] = n
		if !n.Supported {
			continue
		}
		for _, value := range []input.Value{n.Resource.Before, n.Resource.After} {
			for _, attr := range n.Adapter.Identity {
				if s, ok := value.Field(attr).String(); ok && s != "" {
					r.add(s, n)
				}
			}
		}
	}
	return r
}

func (r *resolver) add(key string, n *Node) {
	for _, existing := range r.byValue[key] {
		if existing == n {
			return
		}
	}
	r.byValue[key] = append(r.byValue[key], n)
}

type match struct {
	node     *Node
	evidence model.EvidenceKind
}

type unresolved struct {
	reason model.UnresolvedReason
}

// resolve finds the resources a field refers to. Known values match recorded
// identifiers. Values known only after apply fall back to configuration
// references, and only when those name specific resource instances. Nothing
// is guessed: anything else is reported as unresolved.
func (r *resolver) resolve(from *Node, value input.Value, field string, targets []string) ([]match, []unresolved) {
	var matches []match
	var problems []unresolved
	needConfig := false
	for _, v := range collect(value, strings.Split(field, ".")) {
		switch v.Kind() {
		case input.KindString:
			s, _ := v.String()
			if s == "" {
				continue
			}
			found := filterTypes(r.byValue[s], targets)
			switch len(found) {
			case 0:
				problems = append(problems, unresolved{model.UnresolvedExternal})
			case 1:
				matches = append(matches, match{found[0], model.EvidenceAttribute})
			default:
				problems = append(problems, unresolved{model.UnresolvedAmbiguous})
			}
		case input.KindUnknown:
			needConfig = true
		case input.KindSensitive:
			// A sensitive reference suppresses the relationship: an edge can
			// disclose as much as a value.
			problems = append(problems, unresolved{model.UnresolvedSensitive})
		}
	}
	if needConfig {
		found, referenced := r.fromConfig(from, field, targets)
		// A purely computed value has no configured reference to report;
		// only a reference that cannot be resolved is unresolved.
		if len(found) == 0 && referenced {
			problems = append(problems, unresolved{model.UnresolvedUnknownUntilApply})
		}
		for _, n := range found {
			matches = append(matches, match{n, model.EvidenceConfiguration})
		}
	}
	return dedupe(matches), problems
}

// fromConfig maps configuration references (such as `aws_subnet.private["a"].id`)
// to resource instances. References to a whole collection are ambiguous and
// are ignored.
func (r *resolver) fromConfig(from *Node, field string, targets []string) (found []*Node, referenced bool) {
	var out []*Node
	refs := r.snap.Config.References(from.Resource, field)
	for _, ref := range refs {
		n, ok := r.byAddress[ref]
		if !ok {
			if i := strings.LastIndex(ref, "."); i > 0 {
				n, ok = r.byAddress[ref[:i]]
			}
		}
		if ok && n != from && n.Supported && typeAllowed(n.Resource.Type, targets) {
			out = append(out, n)
		}
	}
	return out, len(refs) > 0
}

// collect walks a dotted path, descending into every element of any list
// on the way (nested blocks and list attributes alike).
func collect(v input.Value, path []string) []input.Value {
	if v.Kind() == input.KindList {
		var out []input.Value
		for _, item := range v.Items() {
			out = append(out, collect(item, path)...)
		}
		return out
	}
	if len(path) == 0 || v.Kind() == input.KindSensitive || v.Kind() == input.KindUnknown {
		return []input.Value{v}
	}
	return collect(v.Field(path[0]), path[1:])
}

func filterTypes(nodes []*Node, targets []string) []*Node {
	var out []*Node
	for _, n := range nodes {
		if typeAllowed(n.Resource.Type, targets) {
			out = append(out, n)
		}
	}
	return out
}

func typeAllowed(typ string, targets []string) bool {
	if len(targets) == 0 {
		return true
	}
	for _, t := range targets {
		if t == typ {
			return true
		}
	}
	return false
}

func dedupe(matches []match) []match {
	seen := map[*Node]bool{}
	var out []match
	for _, m := range matches {
		if !seen[m.node] {
			seen[m.node] = true
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].node.ID < out[j].node.ID })
	return out
}
