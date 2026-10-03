package input

import "strings"

// State format 1.0 is the newest version in the tested fixture matrix.
const knownStateMinor = 0

type stateDocument struct {
	FormatVersion    string `json:"format_version"`
	TerraformVersion string `json:"terraform_version"`
	Values           *struct {
		RootModule stateModule `json:"root_module"`
	} `json:"values"`
}

type stateModule struct {
	Resources    []stateResource `json:"resources"`
	ChildModules []stateModule   `json:"child_modules"`
}

type stateResource struct {
	Address         string `json:"address"`
	Mode            string `json:"mode"`
	Type            string `json:"type"`
	Name            string `json:"name"`
	ProviderName    string `json:"provider_name"`
	Values          any    `json:"values"`
	SensitiveValues any    `json:"sensitive_values"`
}

func readState(data []byte) (*Snapshot, error) {
	var doc stateDocument
	if err := decode(data, &doc); err != nil {
		return nil, err
	}
	notices, err := checkFormat(doc.FormatVersion, knownStateMinor)
	if err != nil {
		return nil, err
	}
	snap := &Snapshot{
		Kind:            KindState,
		Producer:        ProducerUnknown,
		FormatVersion:   doc.FormatVersion,
		ProducerVersion: safeVersion(doc.TerraformVersion),
		Notices:         notices,
	}
	// An empty state exports without values.
	if doc.Values == nil {
		return snap, nil
	}
	var raw []stateResource
	if err := collectState(doc.Values.RootModule, &raw); err != nil {
		return nil, err
	}
	for _, r := range raw {
		snap.Resources = append(snap.Resources, Resource{
			Address:  r.Address,
			Module:   moduleOf(r.Address, r.Mode, r.Type, r.Name),
			Mode:     r.Mode,
			Type:     r.Type,
			Name:     r.Name,
			Provider: providerSource(r.ProviderName),
			After:    buildValue(r.Values, r.SensitiveValues, nil, nil, nil),
			HasAfter: true,
		})
		if snap.Producer == ProducerUnknown {
			snap.Producer = producerFromName(r.ProviderName)
		}
	}
	sortResources(snap.Resources)
	assignProviderKeys(snap)
	return snap, nil
}

func collectState(m stateModule, out *[]stateResource) error {
	for _, r := range m.Resources {
		if len(*out) >= MaxResources {
			return newError(CodeTooManyThings, "The state has more resources than tfviz can process in one report (50,000).")
		}
		*out = append(*out, r)
	}
	for _, child := range m.ChildModules {
		if err := collectState(child, out); err != nil {
			return err
		}
	}
	return nil
}

// moduleOf derives the module instance address from a resource address,
// for example `module.a["x"].aws_s3_bucket.b[0]` gives `module.a["x"]`.
func moduleOf(address, mode, typ, name string) string {
	local := typ + "." + name
	if mode == "data" {
		local = "data." + local
	}
	i := strings.LastIndex(address, "."+local)
	if i < 0 {
		return ""
	}
	return address[:i]
}

// producerFromName infers the producer from a provider's registry host:
// OpenTofu resolves providers from registry.opentofu.org by default.
func producerFromName(name string) Producer {
	switch {
	case strings.HasPrefix(name, "registry.opentofu.org/"):
		return ProducerOpenTofu
	case strings.HasPrefix(name, "registry.terraform.io/"):
		return ProducerTerraform
	}
	return ProducerUnknown
}
