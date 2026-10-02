package mstore

import (
	"slices"
	"testing"

	"github.com/afcollins/kbx/internal/metrics"
)

func TestVisibleFieldsIncludesConstantLabels(t *testing.T) {
	result, err := metrics.ParseFile("../../testdata/collected-metrics-de914d0d-5989-4e9a-990f-9e49c19d9df5/prometheus-ingestionrate.json", 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	store := New()
	store.Load([]*metrics.ParseResult{result})

	visible := store.VisibleFields()
	if !slices.Contains(visible, "job") {
		t.Errorf("VisibleFields() = %v, missing constant job label", visible)
	}
	if !slices.Contains(visible, "namespace") {
		t.Errorf("VisibleFields() = %v, missing constant namespace label", visible)
	}
}
