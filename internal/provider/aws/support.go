package aws

import (
	"fmt"
	"sort"
	"strings"

	"github.com/danushkastanley/tfviz/internal/report/model"
)

var familyTitles = []struct {
	family model.Family
	title  string
}{
	{model.FamilyNetwork, "Networking"},
	{model.FamilySecurity, "Security groups"},
	{model.FamilyLoadBalancing, "Load balancing"},
	{model.FamilyCompute, "Compute"},
	{model.FamilyDatabase, "Databases and caches"},
	{model.FamilyStreaming, "Streaming"},
	{model.FamilyMessaging, "Messaging"},
	{model.FamilyStorage, "Storage"},
	{model.FamilySecrets, "Secrets"},
	{model.FamilyEncryption, "Encryption"},
	{model.FamilyObservability, "Observability"},
	{model.FamilyConfiguration, "Configuration"},
}

// SupportDocument renders docs/support.md from the adapter registry, so the
// published support list can never drift from the code.
func SupportDocument() string {
	var b strings.Builder
	b.WriteString("# Supported resource types\n\n")
	b.WriteString("<!-- Generated from internal/provider/aws. Regenerate with: go test ./internal/provider/aws -update -->\n\n")
	b.WriteString("Supported means the type's placement, approved details and relationships are interpreted and tested against Terraform 1.16 and OpenTofu 1.13 fixtures with AWS provider 6.67. It does not mean every provider attribute is shown: tags, descriptions, policies, secret contents and similar free-form or sensitive values are withheld by design.\n\n")
	b.WriteString("Any other type appears with limited detail and no placement, and `--strict` treats it as a failure.\n")
	for _, ft := range familyTitles {
		var types []string
		for t, a := range registry {
			if a.Family == ft.family {
				types = append(types, t)
			}
		}
		if len(types) == 0 {
			continue
		}
		sort.Strings(types)
		fmt.Fprintf(&b, "\n## %s\n\n| Type | Shown as | Withheld |\n| --- | --- | --- |\n", ft.title)
		for _, t := range types {
			a := registry[t]
			shown := "Resource"
			if a.Role == model.RoleAssociation {
				shown = "Connection between resources"
			}
			if a.DefinesGroup != "" {
				shown = "Container (" + string(a.DefinesGroup) + ")"
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", t, shown, withheld(a))
		}
	}
	return b.String()
}

func withheld(a Adapter) string {
	var names []string
	for _, f := range a.Fields {
		switch {
		case f.Withheld:
			names = append(names, f.Label)
		case f.ChangeOnly:
			names = append(names, f.Label+" (change only)")
		}
	}
	if len(names) == 0 {
		return "—"
	}
	return strings.Join(names, ", ")
}
