package build

import (
	"testing"
	"time"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
	"github.com/danushkastanley/tfviz/internal/testutil"
)

func buildSynthetic(t testing.TB, n int) (*model.Report, time.Duration) {
	data := testutil.SyntheticPlan(n)
	started := time.Now()
	snap, err := input.Read(data, input.KindPlan)
	if err != nil {
		t.Fatal(err)
	}
	report := Report(snap, Options{Source: model.SourceFile, GeneratedAt: time.Unix(0, 0)})
	return report, time.Since(started)
}

// Plan §19: parsing and projection of 500 resources should take under two
// seconds on a developer laptop. The test allows headroom for slow CI.
func TestParsingAndProjectionBudget(t *testing.T) {
	for _, n := range []int{500, 2000, 20000} {
		report, elapsed := buildSynthetic(t, n)
		t.Logf("%d resources, %d relationships: %v", len(report.Resources), len(report.Relationships), elapsed)
		if len(report.Resources) != n || len(report.Relationships) < n {
			t.Fatalf("synthetic stack too small: %d resources, %d relationships", len(report.Resources), len(report.Relationships))
		}
		if n == 500 && elapsed > 2*time.Second {
			t.Errorf("500 resources took %v, budget 2s", elapsed)
		}
	}
}

func BenchmarkReport500(b *testing.B) {
	for b.Loop() {
		buildSynthetic(b, 500)
	}
}

func BenchmarkReport2000(b *testing.B) {
	for b.Loop() {
		buildSynthetic(b, 2000)
	}
}
