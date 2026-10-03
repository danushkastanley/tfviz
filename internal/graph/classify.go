package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// classify labels a subnet public or private only when route evidence
// supports it (plan §9): every route table explicitly associated with the
// subnet must agree. A subnet without an explicit association uses the
// VPC's main route table, which is not evidenced here, so it stays unknown.
func (p *placer) classify(subnet *Node) model.Classification {
	var verdicts []model.Classification
	for _, rel := range p.byTarget[subnet.ID] {
		if rel.Type != model.RelRouteAssociation || !p.current(subnet, rel.Presence) {
			continue
		}
		verdicts = append(verdicts, p.routeTableClass(rel.Source))
	}
	if len(verdicts) == 0 {
		return model.ClassificationUnknown
	}
	for _, v := range verdicts[1:] {
		if v != verdicts[0] {
			return model.ClassificationUnknown
		}
	}
	return verdicts[0]
}

// routeTableClass: public when the table routes to an internet gateway;
// private when its gateway targets are all known and none is an internet
// gateway; unknown otherwise.
func (p *placer) routeTableClass(tableID string) model.Classification {
	table, ok := p.nodes[tableID]
	if !ok {
		return model.ClassificationUnknown
	}
	for _, rel := range p.bySource[tableID] {
		if rel.Type != model.RelRouting || !p.current(table, rel.Presence) {
			continue
		}
		if target, ok := p.nodes[rel.Target]; ok && target.Resource.Type == "aws_internet_gateway" {
			return model.ClassificationPublic
		}
	}
	for _, gateway := range collect(table.current(), []string{"route", "gateway_id"}) {
		switch gateway.Kind() {
		case input.KindUnknown, input.KindSensitive:
			return model.ClassificationUnknown
		case input.KindString:
			if s, _ := gateway.String(); strings.HasPrefix(s, "igw-") {
				return model.ClassificationPublic
			}
		}
	}
	return model.ClassificationPrivate
}

// moduleGroups builds the module view: the root module and every module
// instance, nested by address. IDs continue after the architecture groups.
func moduleGroups(nodes []*Node, offset int) ([]model.Group, map[string]string) {
	instances := map[string]bool{"": true}
	for _, n := range nodes {
		for m := n.Resource.Module; m != ""; m = parentModule(m) {
			instances[m] = true
		}
	}
	addresses := make([]string, 0, len(instances))
	for m := range instances {
		addresses = append(addresses, m)
	}
	sort.Strings(addresses)
	ids := map[string]string{}
	for i, m := range addresses {
		ids[m] = fmt.Sprintf("m%d", offset+i+1)
	}
	groups := make([]model.Group, 0, len(addresses))
	for _, m := range addresses {
		g := model.Group{ID: ids[m], View: model.ViewModules, Kind: model.GroupModule, Label: clipLabel(m), Placement: model.PlacementKnown}
		if m == "" {
			g.Label = "Root module"
		} else {
			g.Parent = ids[parentModule(m)]
		}
		groups = append(groups, g)
	}
	return groups, ids
}

// parentModule drops the last `module.name[key]` segment of an instance address.
func parentModule(instance string) string {
	i := strings.LastIndex(instance, ".module.")
	if i < 0 {
		return ""
	}
	return instance[:i]
}
