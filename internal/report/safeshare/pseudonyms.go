package safeshare

import (
	"fmt"
	"net/netip"
	"strings"
)

// pseudonyms maps original values to stand-ins. The same value always gets
// the same stand-in within one report, so relationships stay readable.
type pseudonyms struct {
	values  map[string]string
	modules map[string]string
	counts  map[string]int
}

func newPseudonyms() *pseudonyms {
	return &pseudonyms{values: map[string]string{}, modules: map[string]string{}, counts: map[string]int{}}
}

func (p *pseudonyms) next(category string) int {
	p.counts[category]++
	return p.counts[category]
}

// value replaces a string with a stand-in shaped like what it replaced.
func (p *pseudonyms) value(s string) string {
	if s == "" {
		return s
	}
	if existing, ok := p.values[s]; ok {
		return existing
	}
	category := classify(s)
	stand := fmt.Sprintf("%s-%d", category, p.next(category))
	if category == "host" {
		stand += ".example"
	}
	p.values[s] = stand
	return stand
}

// module replaces each module name in an instance address, dropping keys.
func (p *pseudonyms) module(address string) string {
	if address == "" {
		return ""
	}
	if existing, ok := p.modules[address]; ok {
		return existing
	}
	var parts []string
	for _, segment := range strings.Split(address, ".module.") {
		name := strings.TrimPrefix(segment, "module.")
		if i := strings.IndexByte(name, '['); i >= 0 {
			name = name[:i]
		}
		key := "module:" + name
		stand, ok := p.values[key]
		if !ok {
			stand = fmt.Sprintf("module_%d", p.next("module"))
			p.values[key] = stand
		}
		parts = append(parts, "module."+stand)
	}
	result := strings.Join(parts, ".")
	p.modules[address] = result
	return result
}

func classify(s string) string {
	switch {
	case strings.HasPrefix(s, "arn:"):
		return "arn"
	case isPrefix(s):
		return "cidr"
	case isAddr(s):
		return "ip"
	case len(s) == 12 && strings.Trim(s, "0123456789") == "":
		return "account"
	case isHost(s):
		return "host"
	default:
		return "name"
	}
}

func isPrefix(s string) bool {
	_, err := netip.ParsePrefix(s)
	return err == nil
}

func isAddr(s string) bool {
	_, err := netip.ParseAddr(s)
	return err == nil
}

// isHost recognises DNS names and host lists such as bootstrap brokers.
func isHost(s string) bool {
	if strings.ContainsAny(s, " /") || !strings.Contains(s, ".") {
		return false
	}
	return strings.ContainsAny(s, "abcdefghijklmnopqrstuvwxyz")
}
