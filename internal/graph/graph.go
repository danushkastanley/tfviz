// Package graph interprets a snapshot as a graph of resources: their
// changes, evidenced relationships and placement. It works only with values
// the reader has already made safe; sensitive values cannot be read here.
package graph

import (
	"fmt"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/provider/aws"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// Node is one resource instance in the graph.
type Node struct {
	ID        string
	Resource  input.Resource
	Adapter   aws.Adapter
	Supported bool
	Change    model.Change
	Label     string
}

// current is the value that describes where a resource is: after the change,
// or before it for resources being destroyed.
func (n *Node) current() input.Value {
	if n.Resource.HasAfter {
		return n.Resource.After
	}
	return n.Resource.Before
}

// Graph is the interpreted snapshot.
type Graph struct {
	Plan          bool
	Nodes         []*Node
	Relationships []model.Relationship
	Unresolved    []model.UnresolvedReference
	Groups        []model.Group
	Placement     map[string]model.ResourceGroups
}

// Build interprets a snapshot. Output order is deterministic: nodes follow
// resource addresses and every derived list is built in node order.
func Build(snap *input.Snapshot) *Graph {
	g := &Graph{Plan: snap.Kind == input.KindPlan, Placement: map[string]model.ResourceGroups{}}
	for i, r := range snap.Resources {
		n := &Node{ID: fmt.Sprintf("r%d", i+1), Resource: r}
		if r.Provider == aws.Provider {
			n.Adapter, n.Supported = aws.Lookup(r.Type)
		}
		n.Change = classify(r, n.Adapter, g.Plan)
		n.Label = label(n)
		g.Nodes = append(g.Nodes, n)
	}
	res := newResolver(snap, g.Nodes)
	g.Relationships, g.Unresolved = buildRelationships(res, g.Nodes, g.Plan)
	g.Groups, g.Placement = place(res, g, snap)
	return g
}

// sides lists the values a node has: before and after for plans (when
// present), the recorded value for state.
func (n *Node) sides(plan bool) []side {
	if !plan {
		return []side{{after: true, value: n.Resource.After}}
	}
	var out []side
	if n.Resource.HasBefore {
		out = append(out, side{after: false, value: n.Resource.Before})
	}
	if n.Resource.HasAfter {
		out = append(out, side{after: true, value: n.Resource.After})
	}
	return out
}

type side struct {
	after bool
	value input.Value
}
