package pipeline

import (
	"elcom/internal"
	"elcom/internal/util"
)

type NormalizedItem struct {
	internal.ExtractionItem
	NormalizedNameOrCode string
}

func NormalizeItems(items []internal.ExtractionItem) []NormalizedItem {
	out := make([]NormalizedItem, 0, len(items))
	for _, item := range items {
		source := item.RawLine
		if item.NameOrCode != nil {
			source = *item.NameOrCode
		}
		out = append(out, NormalizedItem{
			ExtractionItem:       item,
			NormalizedNameOrCode: util.NormalizeHeader(source),
		})
	}
	return out
}
