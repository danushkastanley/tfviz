package html

import (
	"github.com/danushkastanley/tfviz/internal/report/model"
)

// summaryView is the static, script-free fallback. It keeps the report useful
// when JavaScript is disabled or blocked; the interface replaces it on load.
type summaryView struct {
	Mode    string
	Counts  []summaryCount
	Changes []summaryRow
	Total   int
}

type summaryCount struct {
	Label string
	Count int
}

type summaryRow struct {
	Change  string
	Address string
	Type    string
}

var actionLabels = map[model.Action]string{
	model.ActionCreate:      "Added",
	model.ActionUpdate:      "Modified",
	model.ActionDelete:      "Destroyed",
	model.ActionReplace:     "Replaced",
	model.ActionRead:        "Read during apply",
	model.ActionUnsupported: "Unsupported action",
	model.ActionNoOp:        "Unchanged",
}

func summarise(report *model.Report) summaryView {
	view := summaryView{Total: len(report.Resources), Mode: "Recorded state"}
	if report.Mode == model.ModePlan {
		view.Mode = "Plan review"
		s := report.Summary
		for _, c := range []summaryCount{
			{actionLabels[model.ActionCreate], s.Create},
			{actionLabels[model.ActionUpdate], s.Update},
			{actionLabels[model.ActionReplace], s.Replace},
			{actionLabels[model.ActionDelete], s.Delete},
			{actionLabels[model.ActionRead], s.Read},
			{actionLabels[model.ActionUnsupported], s.Unsupported},
		} {
			if c.Count > 0 {
				view.Counts = append(view.Counts, c)
			}
		}
	}
	for _, r := range report.Resources {
		if r.Change.Action == model.ActionNoOp {
			continue
		}
		view.Changes = append(view.Changes, summaryRow{Change: actionLabels[r.Change.Action], Address: r.Address, Type: r.Type})
	}
	return view
}
