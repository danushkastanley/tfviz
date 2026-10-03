package graph

import (
	"fmt"
	"sort"

	"github.com/danushkastanley/tfviz/internal/provider/aws"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

type relKey struct {
	source, target, via string
	typ                 model.RelationshipType
	field               string
}

type relSeen struct {
	before, after bool
	evidence      model.EvidenceKind
	order         int
}

type relCollector struct {
	plan       bool
	seen       map[relKey]*relSeen
	unresolved map[model.UnresolvedReference]bool
	order      []model.UnresolvedReference
}

func (c *relCollector) add(k relKey, after bool, evidence model.EvidenceKind) {
	if k.source == k.target {
		return
	}
	s, ok := c.seen[k]
	if !ok {
		s = &relSeen{order: len(c.seen)}
		c.seen[k] = s
	}
	if after {
		s.after = true
		s.evidence = evidence // the post-change evidence describes the result
	} else {
		s.before = true
		if s.evidence == "" {
			s.evidence = evidence
		}
	}
}

func (c *relCollector) miss(n *Node, typ model.RelationshipType, field string, problems []unresolved) {
	for _, p := range problems {
		u := model.UnresolvedReference{Source: n.ID, Type: typ, Field: field, Reason: p.reason}
		if !c.unresolved[u] {
			c.unresolved[u] = true
			c.order = append(c.order, u)
		}
	}
}

// buildRelationships evaluates every adapter rule on every side of every
// supported resource, then merges the sides into one relationship with a
// presence of before, after or both.
func buildRelationships(res *resolver, nodes []*Node, plan bool) ([]model.Relationship, []model.UnresolvedReference) {
	c := &relCollector{plan: plan, seen: map[relKey]*relSeen{}, unresolved: map[model.UnresolvedReference]bool{}}
	type inherited struct {
		node  *Node
		rule  aws.Relation
		after bool
		via   match
	}
	var through []inherited

	for _, n := range nodes {
		if !n.Supported {
			continue
		}
		for _, s := range n.sides(plan) {
			for _, rule := range n.Adapter.Relations {
				targets, problems := res.resolve(n, s.value, rule.Field, rule.Targets)
				c.miss(n, rule.Type, rule.Field, problems)
				if rule.Through {
					for _, t := range targets {
						through = append(through, inherited{n, rule, s.after, t})
					}
					continue
				}
				sources := []match{{node: n, evidence: model.EvidenceAttribute}}
				via := ""
				if rule.From != "" {
					var fromProblems []unresolved
					sources, fromProblems = res.resolve(n, s.value, rule.From, rule.FromTargets)
					c.miss(n, rule.Type, rule.From, fromProblems)
					via = n.ID
				}
				for _, src := range sources {
					for _, t := range targets {
						kind := weakest(src.evidence, t.evidence)
						if via != "" && kind == model.EvidenceAttribute {
							kind = model.EvidenceAssociationResource
						}
						c.add(relKey{src.node.ID, t.node.ID, via, rule.Type, rule.Field}, s.after, kind)
					}
				}
			}
		}
	}

	// Inherited relationships, such as a DB instance's subnets through its
	// DB subnet group, reuse the intermediate resource's own evidence.
	// Candidates are gathered in insertion order before any are added, so the
	// result never depends on map iteration order.
	direct := c.sortedKeys()
	for _, in := range through {
		for _, k := range direct {
			s := c.seen[k]
			if k.source != in.via.node.ID || k.typ != in.rule.Type || k.via != "" {
				continue
			}
			if (in.after && !s.after) || (!in.after && !s.before) {
				continue
			}
			field := in.rule.Field + "." + k.field
			c.add(relKey{in.node.ID, k.target, in.via.node.ID, in.rule.Type, field}, in.after, weakest(in.via.evidence, s.evidence))
		}
	}
	return c.result()
}

// weakest picks the less direct evidence of two hops: configuration beats
// nothing but never upgrades to a recorded attribute.
func weakest(a, b model.EvidenceKind) model.EvidenceKind {
	if a == model.EvidenceConfiguration || b == model.EvidenceConfiguration {
		return model.EvidenceConfiguration
	}
	return a
}

func (c *relCollector) sortedKeys() []relKey {
	keys := make([]relKey, 0, len(c.seen))
	for k := range c.seen {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return c.seen[keys[i]].order < c.seen[keys[j]].order })
	return keys
}

func (c *relCollector) result() ([]model.Relationship, []model.UnresolvedReference) {
	keys := c.sortedKeys()
	rels := make([]model.Relationship, 0, len(keys))
	for i, k := range keys {
		s := c.seen[k]
		presence := model.PresenceBoth
		switch {
		case !c.plan:
		case s.before && !s.after:
			presence = model.PresenceBefore
		case s.after && !s.before:
			presence = model.PresenceAfter
		}
		rels = append(rels, model.Relationship{
			ID: fmt.Sprintf("e%d", i+1), Source: k.source, Target: k.target, Type: k.typ, Presence: presence,
			Evidence: model.Evidence{Kind: s.evidence, Field: k.field, Via: k.via},
		})
	}
	return rels, c.order
}
