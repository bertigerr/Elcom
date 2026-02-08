package pipeline

import (
	"testing"

	"elcom/internal"
	"elcom/internal/config"
	"elcom/internal/util"
)

func sp(v string) *string { return &v }

func TestMatcher(t *testing.T) {
	products := []internal.ProductRecord{
		{ID: 1, SyncUID: sp("sync-1"), Header: "Кабель ВВГнг 3x2.5", Articul: sp("ELC0100203802")},
		{ID: 2, SyncUID: sp("sync-2"), Header: "Кабель ВВГнг 3x4", Articul: sp("ELC0100203803")},
	}
	cfg, _ := config.Load()
	m := NewMatcher(cfg, products)

	qty := 2.0
	item := NormalizedItem{ExtractionItem: internal.ExtractionItem{LineNo: 1, Source: internal.SourceEmailText, RawLine: "ELC0100203802 2 шт", NameOrCode: sp("ELC0100203802"), Qty: &qty}, NormalizedNameOrCode: util.NormalizeHeader("ELC0100203802")}
	res := m.Match(item)
	if res.Status != internal.MatchOK || res.Reason != internal.ReasonCode || res.Product == nil || res.Product.ID == nil || *res.Product.ID != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
}
