package graph

import (
	"strings"
	"unicode/utf8"

	"github.com/danushkastanley/tfviz/internal/input"
)

const maxLabel = 256

// label picks an approved display name: the adapter's label sources (such
// as the Name tag or the resource's name attribute), else a fallback built
// from the configuration name and instance key.
func label(n *Node) string {
	value := n.current()
	for _, path := range n.Adapter.LabelFrom {
		if s, ok := value.Path(strings.Split(path, ".")...).String(); ok && strings.TrimSpace(s) != "" {
			return clipLabel(s)
		}
	}
	if !n.Supported {
		return clipLabel(n.Resource.Address)
	}
	local := n.Resource.Name + instanceKey(n.Resource)
	if n.Adapter.Noun == "" {
		return clipLabel(local)
	}
	return clipLabel(n.Adapter.Noun + " " + local)
}

// instanceKey returns the `["a"]` or `[0]` suffix of a resource address.
func instanceKey(r input.Resource) string {
	local := r.Type + "." + r.Name
	if r.Mode == "data" {
		local = "data." + local
	}
	i := strings.LastIndex(r.Address, local)
	if i < 0 {
		return ""
	}
	return r.Address[i+len(local):]
}

func clipLabel(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
	if len(s) <= maxLabel {
		return s
	}
	cut := maxLabel - len("…")
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
