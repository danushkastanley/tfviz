// Package aws declares how supported AWS resource types are interpreted:
// their approved metadata, how they are placed and which recorded attributes
// evidence their relationships. Types without an adapter get a limited
// generic card.
package aws

import (
	"github.com/danushkastanley/tfviz/internal/projection"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// Provider is the provider source these adapters interpret.
const Provider = "hashicorp/aws"

// Adapter describes one resource type.
type Adapter struct {
	Family model.Family
	Role   model.Role
	// Noun names the resource kind in fallback labels, e.g. "NAT gateway".
	Noun string
	// LabelFrom lists value paths tried in order for the display label.
	LabelFrom []string
	// Identity lists attributes other resources use to refer to this one.
	Identity []string
	// DefinesGroup makes the resource a container in the architecture view.
	DefinesGroup model.GroupKind
	Placement    Placement
	Fields       []projection.Field
	Relations    []Relation
}

// Placement says where a resource sits in the architecture view.
type Placement struct {
	// VPCField names an attribute holding the VPC ID.
	VPCField string
	// SubnetHome places a resource in a subnet when it belongs to exactly one.
	SubnetHome bool
	// Regional resources sit with the region's services, outside any VPC.
	Regional bool
}

// Relation evidences a relationship from a recorded attribute.
type Relation struct {
	Type model.RelationshipType
	// Field is a dotted path; lists and nested block lists are walked.
	Field string
	// Targets restricts matches to these resource types.
	Targets []string
	// From makes this an association: the relationship runs from the
	// resource referenced by From to the one referenced by Field, via this
	// resource. Empty means the relationship starts at this resource.
	From        string
	FromTargets []string
	// Through follows the matched target's own relation of this type, e.g. a
	// DB instance's subnets through its DB subnet group.
	Through bool
}

var registry = map[string]Adapter{}

func register(resourceType string, a Adapter) {
	if a.Role == "" {
		a.Role = model.RoleEntity
	}
	if len(a.Identity) == 0 {
		a.Identity = []string{"id", "arn"}
	}
	if len(a.LabelFrom) == 0 {
		a.LabelFrom = []string{"tags.Name", "name"}
	}
	registry[resourceType] = a
}

// Lookup returns the adapter for a resource type.
func Lookup(resourceType string) (Adapter, bool) {
	a, ok := registry[resourceType]
	return a, ok
}

// Types lists every supported resource type.
func Types() []string {
	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}

func init() {
	registerNetwork()
	registerSecurity()
	registerWorkloads()
	registerSupporting()
}
