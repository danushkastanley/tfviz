package graph

import (
	"strings"

	"github.com/danushkastanley/tfviz/internal/input"
)

// location is where a resource's account and region are recorded.
type location struct {
	account string
	region  string
}

// locator infers account and region from explicit evidence: a resource's
// own ARN, owner or availability zone, then its provider configuration (one
// provider configuration operates in one account and region).
type locator struct {
	snap            *input.Snapshot
	providerAccount map[string]string
	providerRegion  map[string]string
}

func newLocator(snap *input.Snapshot, nodes []*Node) *locator {
	l := &locator{snap: snap, providerAccount: map[string]string{}, providerRegion: map[string]string{}}
	accounts := map[string]map[string]bool{}
	regions := map[string]map[string]bool{}
	for _, n := range nodes {
		own := l.own(n)
		key := n.Resource.ProviderKey
		if own.account != "" {
			addTo(accounts, key, own.account)
		}
		if own.region != "" {
			addTo(regions, key, own.region)
		}
	}
	for key, set := range accounts {
		if len(set) == 1 {
			l.providerAccount[key] = only(set)
		}
	}
	for key, set := range regions {
		if len(set) == 1 {
			l.providerRegion[key] = only(set)
		}
	}
	return l
}

func (l *locator) locate(n *Node) location {
	loc := l.own(n)
	key := n.Resource.ProviderKey
	if loc.account == "" {
		loc.account = l.providerAccount[key]
	}
	if loc.region == "" {
		loc.region = l.snap.Regions[key]
	}
	if loc.region == "" {
		loc.region = l.providerRegion[key]
	}
	return loc
}

func (l *locator) own(n *Node) location {
	v := n.current()
	var loc location
	if arn, ok := v.Field("arn").String(); ok {
		loc.region, loc.account = parseARN(arn)
	}
	if loc.account == "" {
		if owner, ok := v.Field("owner_id").String(); ok && isAccountID(owner) {
			loc.account = owner
		}
	}
	if loc.region == "" {
		if az, ok := v.Field("availability_zone").String(); ok && len(az) > 1 {
			loc.region = strings.TrimRight(az, "abcdefghijklmnopqrstuvwxyz")
			if loc.region == az {
				loc.region = ""
			}
		}
	}
	return loc
}

// parseARN extracts region and account from arn:partition:service:region:account:resource.
func parseARN(arn string) (region, account string) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 || parts[0] != "arn" {
		return "", ""
	}
	if isAccountID(parts[4]) {
		account = parts[4]
	}
	return parts[3], account
}

func isAccountID(s string) bool {
	if len(s) != 12 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func addTo(m map[string]map[string]bool, key, value string) {
	if m[key] == nil {
		m[key] = map[string]bool{}
	}
	m[key][value] = true
}

func only(set map[string]bool) string {
	for v := range set {
		return v
	}
	return ""
}
