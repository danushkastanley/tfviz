package model

type RelationshipType string

const (
	RelSubnetMembership        RelationshipType = "subnet_membership"
	RelSecurityGroupAttachment RelationshipType = "security_group_attachment"
	RelSecurityGroupRule       RelationshipType = "security_group_rule"
	RelRouting                 RelationshipType = "routing"
	RelRouteAssociation        RelationshipType = "route_association"
	RelServiceReference        RelationshipType = "service_reference"
	RelSecretAssociation       RelationshipType = "secret_association"
	RelEncryptionKey           RelationshipType = "encryption_key"
	RelLogDestination          RelationshipType = "log_destination"
)

type Presence string

const (
	PresenceBefore Presence = "before"
	PresenceAfter  Presence = "after"
	PresenceBoth   Presence = "both"
)

type EvidenceKind string

const (
	EvidenceAttribute           EvidenceKind = "attribute"
	EvidenceConfiguration       EvidenceKind = "configuration"
	EvidenceAssociationResource EvidenceKind = "association_resource"
)

// Relationship records an association evidenced by the input. It describes
// configuration, never observed traffic or proven reachability.
type Relationship struct {
	ID       string           `json:"id"`
	Source   string           `json:"source"`
	Target   string           `json:"target"`
	Type     RelationshipType `json:"type"`
	Presence Presence         `json:"presence"`
	Evidence Evidence         `json:"evidence"`
}

type Evidence struct {
	Kind  EvidenceKind `json:"kind"`
	Field string       `json:"field"`
	Via   string       `json:"via,omitempty"`
}

type UnresolvedReason string

const (
	UnresolvedUnknownUntilApply UnresolvedReason = "unknown_until_apply"
	UnresolvedExternal          UnresolvedReason = "external"
	UnresolvedAmbiguous         UnresolvedReason = "ambiguous"
	UnresolvedSensitive         UnresolvedReason = "sensitive"
)

// UnresolvedReference keeps a reference visible without inventing a target.
type UnresolvedReference struct {
	Source string           `json:"source"`
	Type   RelationshipType `json:"type"`
	Field  string           `json:"field"`
	Reason UnresolvedReason `json:"reason"`
}

type View string

const (
	ViewArchitecture View = "architecture"
	ViewModules      View = "modules"
)

type GroupKind string

const (
	GroupAccount          GroupKind = "account"
	GroupRegion           GroupKind = "region"
	GroupVPC              GroupKind = "vpc"
	GroupAvailabilityZone GroupKind = "availability_zone"
	GroupSubnet           GroupKind = "subnet"
	GroupRegionalServices GroupKind = "regional_services"
	GroupGlobalServices   GroupKind = "global_services"
	GroupUnplaced         GroupKind = "unplaced"
	GroupModule           GroupKind = "module"
)

type Placement string

const (
	PlacementKnown      Placement = "known"
	PlacementUnresolved Placement = "unresolved"
)

type Classification string

const (
	ClassificationPublic  Classification = "public"
	ClassificationPrivate Classification = "private"
	ClassificationUnknown Classification = "unknown"
)

type Group struct {
	ID             string         `json:"id"`
	View           View           `json:"view"`
	Kind           GroupKind      `json:"kind"`
	Label          string         `json:"label"`
	Parent         string         `json:"parent,omitempty"`
	Placement      Placement      `json:"placement"`
	Resource       string         `json:"resource,omitempty"`
	Classification Classification `json:"classification,omitempty"`
}
