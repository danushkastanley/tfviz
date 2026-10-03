package graph

import (
	"fmt"
	"sort"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// container is a logical architecture group before IDs are assigned.
type container struct {
	key            string
	kind           model.GroupKind
	label          string
	parent         string // parent container key
	resource       *Node
	placement      model.Placement
	classification model.Classification
}

type placer struct {
	g          *Graph
	res        *resolver
	loc        *locator
	containers map[string]*container
	nodeIn     map[string]string // node ID → container key
}

// place builds the architecture and module groups and assigns every
// resource a primary group in each view.
func place(res *resolver, g *Graph, snap *input.Snapshot) ([]model.Group, map[string]model.ResourceGroups) {
	p := &placer{g: g, res: res, loc: newLocator(snap, g.Nodes), containers: map[string]*container{}, nodeIn: map[string]string{}}
	// VPCs first, then subnets, so other resources can find their containers.
	for _, n := range g.Nodes {
		if n.Adapter.DefinesGroup == model.GroupVPC {
			p.addVPC(n)
		}
	}
	for _, n := range g.Nodes {
		if n.Adapter.DefinesGroup == model.GroupSubnet {
			p.addSubnet(n)
		}
	}
	// Resources that follow another are placed last, beside their anchor.
	for _, n := range g.Nodes {
		if n.Adapter.DefinesGroup == "" && n.Adapter.Placement.Follow == "" {
			p.nodeIn[n.ID] = p.home(n)
		}
	}
	for _, n := range g.Nodes {
		if n.Adapter.DefinesGroup == "" && n.Adapter.Placement.Follow != "" {
			p.nodeIn[n.ID] = p.follow(n)
		}
	}
	groups, ids := p.emit()
	modules, moduleIDs := moduleGroups(g.Nodes, len(groups))
	placement := map[string]model.ResourceGroups{}
	for _, n := range g.Nodes {
		placement[n.ID] = model.ResourceGroups{Architecture: ids[p.nodeIn[n.ID]], Modules: moduleIDs[n.Resource.Module]}
	}
	return append(groups, modules...), placement
}

func (p *placer) ensure(c container) string {
	if _, ok := p.containers[c.key]; !ok {
		p.containers[c.key] = &c
	}
	return c.key
}

func (p *placer) region(loc location) string {
	account := p.ensure(container{key: "account|" + loc.account, kind: model.GroupAccount, label: accountLabel(loc.account), placement: known(loc.account)})
	return p.ensure(container{key: "region|" + loc.account + "|" + loc.region, kind: model.GroupRegion, label: regionLabel(loc.region), parent: account, placement: known(loc.region)})
}

func (p *placer) addVPC(n *Node) {
	region := p.region(p.loc.locate(n))
	p.nodeIn[n.ID] = region
	p.ensure(container{key: "vpc|" + n.ID, kind: model.GroupVPC, label: n.Label, parent: region, resource: n, placement: model.PlacementKnown})
}

func (p *placer) addSubnet(n *Node) {
	parent := p.subnetParent(n)
	p.nodeIn[n.ID] = parent
	label := n.Label
	if az, ok := n.current().Field("availability_zone").String(); ok && az != "" {
		label += " · " + az
	}
	p.ensure(container{key: "subnet|" + n.ID, kind: model.GroupSubnet, label: label, parent: parent, resource: n,
		placement: model.PlacementKnown, classification: p.classify(n)})
}

// vpcOf finds the VPC container for a resource's vpc_id, creating a group
// for a VPC managed elsewhere. It reports false when the VPC cannot be
// established.
func (p *placer) vpcOf(n *Node) (string, bool) {
	region := p.region(p.loc.locate(n))
	matches, _ := p.res.resolve(n, n.current(), "vpc_id", []string{"aws_vpc"})
	if len(matches) == 1 {
		return "vpc|" + matches[0].node.ID, true
	}
	if id, ok := n.current().Field("vpc_id").String(); ok && id != "" {
		return p.ensure(container{key: "extvpc|" + region + "|" + id, kind: model.GroupVPC,
			label: "VPC " + clipLabel(id) + " (outside this report)", parent: region, placement: model.PlacementKnown}), true
	}
	return region, false
}

// subnetParent is the VPC container for a subnet, or an unresolved VPC
// group when the subnet's VPC cannot be established.
func (p *placer) subnetParent(n *Node) string {
	if key, ok := p.vpcOf(n); ok {
		return key
	}
	region := p.region(p.loc.locate(n))
	return p.ensure(container{key: "novpc|" + region, kind: model.GroupVPC, label: "VPC not established", parent: region, placement: model.PlacementUnresolved})
}

// home places a resource that does not define a group (plan §9): in its
// subnet when it lives in exactly one, in the VPC of its subnets or vpc_id,
// with regional services, or apart when placement cannot be established.
func (p *placer) home(n *Node) string {
	if !n.Supported {
		return p.unplaced()
	}
	subnets := p.subnetsOf(n)
	if len(subnets) == 1 && n.Adapter.Placement.SubnetHome {
		return "subnet|" + subnets[0]
	}
	if len(subnets) > 0 {
		parents := map[string]bool{}
		for _, s := range subnets {
			parents[p.containers["subnet|"+s].parent] = true
		}
		if len(parents) == 1 {
			return only(parents)
		}
		return p.region(p.loc.locate(n))
	}
	if n.Adapter.Placement.VPCField != "" {
		if key, ok := p.vpcOf(n); ok {
			return key
		}
		return p.unplaced()
	}
	if n.Adapter.Placement.Regional {
		loc := p.loc.locate(n)
		region := p.region(loc)
		return p.ensure(container{key: "services|" + region, kind: model.GroupRegionalServices,
			label: "Regional services · " + regionLabel(loc.region), parent: region, placement: known(loc.region)})
	}
	return p.unplaced()
}

// follow places a resource beside the resources it references: with a
// single anchor, in the anchor's group (or the group it defines); with
// several, in the container they share.
func (p *placer) follow(n *Node) string {
	place := n.Adapter.Placement
	matches, _ := p.res.resolve(n, n.current(), place.Follow, place.FollowTargets)
	homes := map[string]bool{}
	for _, m := range matches {
		if m.node.Adapter.DefinesGroup != "" && len(matches) == 1 {
			homes[string(m.node.Adapter.DefinesGroup)+"|"+m.node.ID] = true
			continue
		}
		// For several subnets this is their shared VPC.
		homes[p.nodeIn[m.node.ID]] = true
	}
	if len(homes) == 1 {
		if home := only(homes); home != "" {
			return home
		}
	}
	return p.unplaced()
}

func (p *placer) unplaced() string {
	return p.ensure(container{key: "unplaced", kind: model.GroupUnplaced, label: "Placement not established", placement: model.PlacementUnresolved})
}

// subnetsOf lists subnets a resource belongs to on its current side.
func (p *placer) subnetsOf(n *Node) []string {
	var out []string
	for _, rel := range p.g.Relationships {
		if rel.Source == n.ID && rel.Type == model.RelSubnetMembership && p.current(n, rel.Presence) {
			if _, ok := p.containers["subnet|"+rel.Target]; ok {
				out = append(out, rel.Target)
			}
		}
	}
	return out
}

// current reports whether a relationship exists on the node's current side.
func (p *placer) current(n *Node, presence model.Presence) bool {
	if n.Resource.HasAfter || !p.g.Plan {
		return presence != model.PresenceBefore
	}
	return presence != model.PresenceAfter
}

// emit assigns group IDs in a deterministic order: accounts and regions by
// name, VPCs and subnets by resource order, then services and unplaced.
func (p *placer) emit() ([]model.Group, map[string]string) {
	keys := make([]string, 0, len(p.containers))
	for k := range p.containers {
		keys = append(keys, k)
	}
	rank := map[model.GroupKind]int{model.GroupAccount: 0, model.GroupRegion: 1, model.GroupVPC: 2, model.GroupSubnet: 3, model.GroupRegionalServices: 4, model.GroupUnplaced: 5}
	sortKey := func(c *container) string {
		if c.resource != nil {
			return fmt.Sprintf("%06d", nodeIndex(c.resource))
		}
		return c.key
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := p.containers[keys[i]], p.containers[keys[j]]
		if rank[a.kind] != rank[b.kind] {
			return rank[a.kind] < rank[b.kind]
		}
		return sortKey(a) < sortKey(b)
	})
	ids := map[string]string{}
	for i, k := range keys {
		ids[k] = fmt.Sprintf("g%d", i+1)
	}
	groups := make([]model.Group, 0, len(keys))
	for _, k := range keys {
		c := p.containers[k]
		group := model.Group{ID: ids[k], View: model.ViewArchitecture, Kind: c.kind, Label: clipLabel(c.label),
			Parent: ids[c.parent], Placement: c.placement, Classification: c.classification}
		if c.resource != nil {
			group.Resource = c.resource.ID
		}
		groups = append(groups, group)
	}
	return groups, ids
}

func nodeIndex(n *Node) int {
	var i int
	_, _ = fmt.Sscanf(n.ID, "r%d", &i)
	return i
}

func known(value string) model.Placement {
	if value == "" {
		return model.PlacementUnresolved
	}
	return model.PlacementKnown
}

func accountLabel(account string) string {
	if account == "" {
		return "Account not recorded"
	}
	return "Account " + account
}

func regionLabel(region string) string {
	if region == "" {
		return "Region not recorded"
	}
	return region
}
