package catalog

import (
	"elcom/internal"
	"elcom/internal/util"
)

type Index struct {
	ProductsByID         map[int]internal.ProductRecord
	ByCode               map[string][]internal.ProductRecord
	ByHeader             map[string][]internal.ProductRecord
	TokenToProductIDs    map[string]map[int]struct{}
	NormalizedHeaderByID map[int]string
}

func BuildIndex(products []internal.ProductRecord) *Index {
	idx := &Index{
		ProductsByID:         map[int]internal.ProductRecord{},
		ByCode:               map[string][]internal.ProductRecord{},
		ByHeader:             map[string][]internal.ProductRecord{},
		TokenToProductIDs:    map[string]map[int]struct{}{},
		NormalizedHeaderByID: map[int]string{},
	}

	for _, p := range products {
		idx.ProductsByID[p.ID] = p
		normHeader := util.NormalizeHeader(p.Header)
		idx.NormalizedHeaderByID[p.ID] = normHeader
		idx.ByHeader[normHeader] = append(idx.ByHeader[normHeader], p)

		addCode := func(code *string) {
			if code == nil {
				return
			}
			norm := util.NormalizeCode(*code)
			if norm == "" {
				return
			}
			idx.ByCode[norm] = append(idx.ByCode[norm], p)
		}

		addCode(p.Articul)
		addCode(p.SyncUID)
		addCode(p.FlatCodes.Elcom)
		addCode(p.FlatCodes.Manufacturer)
		addCode(p.FlatCodes.Raec)
		addCode(p.FlatCodes.PC)
		addCode(p.FlatCodes.Etm)
		for _, analog := range p.AnalogCodes {
			ac := analog
			addCode(&ac)
		}

		for _, token := range util.Tokenize(p.Header) {
			if _, ok := idx.TokenToProductIDs[token]; !ok {
				idx.TokenToProductIDs[token] = map[int]struct{}{}
			}
			idx.TokenToProductIDs[token][p.ID] = struct{}{}
		}
	}

	return idx
}
