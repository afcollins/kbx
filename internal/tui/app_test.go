package tui

import (
	"testing"

	"github.com/afcollins/kbx/internal/metrics"
	"github.com/afcollins/kbx/internal/mstore"
	"github.com/afcollins/kbx/internal/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateMetricsSizesWithoutFacetsSizesHeatmap(t *testing.T) {
	model := NewModel(nil)
	model.width = 120
	model.height = 40
	model.metricStore = mstore.New()
	model.metricStore.Load([]*metrics.ParseResult{{
		Events: []metrics.MetricEvent{{MetricName: "metric", UUID: "id", JobName: "job"}},
	}})
	model.buildMetricFacets()
	if model.mTotal != 0 {
		t.Fatalf("expected no visible fields, got %d", model.mTotal)
	}

	model.updateMetricsSizes()

	if model.scatter.Width != model.width {
		t.Errorf("heatmap width = %d, want %d", model.scatter.Width, model.width)
	}
	if model.scatter.Height == 0 {
		t.Error("heatmap height was not initialized")
	}
}

func TestMetricsHeatmapCanBeMaximized(t *testing.T) {
	model := NewModel(nil)
	model.width = 120
	model.height = 40
	model.metricStore = mstore.New()
	model.metricStore.Load([]*metrics.ParseResult{{
		Events: []metrics.MetricEvent{{MetricName: "metric", UUID: "id", JobName: "job", Value: 1}},
	}})
	model.buildMetricFacets()
	model.updateMetricsSizes()
	model.setMetricsFocus(model.mTotal) // heatmap

	updated, _ := model.handleMetricsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	maximized := updated.(Model)
	if !maximized.maximized {
		t.Fatal("heatmap was not maximized")
	}

	_ = maximized.metricsDashboardView()
	if maximized.scatter.Width != maximized.width {
		t.Errorf("maximized heatmap width = %d, want %d", maximized.scatter.Width, maximized.width)
	}
	if maximized.scatter.Height != maximized.height-styles.StatusBarHeight {
		t.Errorf("maximized heatmap height = %d, want %d", maximized.scatter.Height, maximized.height-styles.StatusBarHeight)
	}
}
