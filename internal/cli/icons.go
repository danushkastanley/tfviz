package cli

import (
	"github.com/danushkastanley/tfviz/internal/icons"
	"github.com/danushkastanley/tfviz/internal/provider/aws"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

const iconsHelp = "folder of the official AWS Architecture Icons pack you downloaded; its icons are embedded in the report"

// loadIcons embeds icons from the user's pack for the resource types the
// report contains.
func loadIcons(dir string, r *model.Report) (*model.Icons, error) {
	seen := map[string]bool{}
	var types []string
	for _, res := range r.Resources {
		if !seen[res.Type] {
			seen[res.Type] = true
			types = append(types, res.Type)
		}
	}
	set, err := icons.Load(dir, aws.IconStems(), types)
	if err != nil {
		return nil, &usageError{err.Error()}
	}
	return set, nil
}
