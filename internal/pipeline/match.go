package pipeline

import (
	"sort"

	"elcom/internal"
	"elcom/internal/catalog"
	"elcom/internal/config"
	"elcom/internal/util"
)

type Matcher struct {
	cfg   config.Config
	index *catalog.Index
}

func NewMatcher(cfg config.Config, products []internal.ProductRecord) *Matcher {
	return &Matcher{cfg: cfg, index: catalog.BuildIndex(products)}
}

func (m *Matcher) Match(item NormalizedItem) internal.MatchResult {
	normalized := item.NormalizedNameOrCode
	if normalized == "" {
		normalized = util.NormalizeHeader(item.RawLine)
	}

	nameOrCode := ""
	if item.NameOrCode != nil {
		nameOrCode = *item.NameOrCode
	}
	codeCandidate := util.NormalizeCode(nameOrCode)

	if util.LooksLikeCode(nameOrCode) && codeCandidate != "" {
		byCode := m.index.ByCode[codeCandidate]
		if len(byCode) == 1 {
			result := internal.MatchResult{
				Status:     internal.MatchOK,
				Confidence: 0.99,
				Reason:     internal.ReasonCode,
				Product:    toMatchProduct(byCode[0]),
				Candidates: []internal.MatchCandidate{{ID: byCode[0].ID, SyncUID: byCode[0].SyncUID, Header: byCode[0].Header, Score: 0.99}},
			}
			return m.adjustForInvalidQty(item, result)
		}
		if len(byCode) > 1 {
			return internal.MatchResult{
				Status:     internal.MatchReview,
				Confidence: 0.80,
				Reason:     internal.ReasonCode,
				Product:    nil,
				Candidates: toCandidates(byCode, 0.80),
			}
		}
	}

	exact := m.index.ByHeader[normalized]
	if len(exact) == 1 {
		result := internal.MatchResult{
			Status:     internal.MatchOK,
			Confidence: 0.95,
			Reason:     internal.ReasonHeader,
			Product:    toMatchProduct(exact[0]),
			Candidates: []internal.MatchCandidate{{ID: exact[0].ID, SyncUID: exact[0].SyncUID, Header: exact[0].Header, Score: 0.95}},
		}
		return m.adjustForInvalidQty(item, result)
	}
	if len(exact) > 1 {
		return internal.MatchResult{
			Status:     internal.MatchReview,
			Confidence: 0.78,
			Reason:     internal.ReasonHeader,
			Product:    nil,
			Candidates: toCandidates(exact, 0.78),
		}
	}

	candidates := m.rankCandidates(normalized)
	if len(candidates) == 0 {
		return internal.MatchResult{Status: internal.MatchNotFound, Confidence: 0, Reason: internal.ReasonNone, Product: nil, Candidates: []internal.MatchCandidate{}}
	}

	top1 := candidates[0]
	gap := top1.Score
	if len(candidates) > 1 {
		gap = top1.Score - candidates[1].Score
	}

	best := m.index.ProductsByID[top1.ID]
	var result internal.MatchResult
	if top1.Score >= m.cfg.MatchOKThreshold && gap >= m.cfg.MatchGapThreshold {
		result = internal.MatchResult{Status: internal.MatchOK, Confidence: top1.Score, Reason: internal.ReasonFuzzy, Product: toMatchProduct(best), Candidates: candidates}
	} else if top1.Score >= m.cfg.MatchReviewThreshold {
		result = internal.MatchResult{Status: internal.MatchReview, Confidence: top1.Score, Reason: internal.ReasonFuzzy, Product: toMatchProduct(best), Candidates: candidates}
	} else {
		result = internal.MatchResult{Status: internal.MatchNotFound, Confidence: top1.Score, Reason: internal.ReasonNone, Product: nil, Candidates: candidates}
	}

	return m.adjustForInvalidQty(item, result)
}

func (m *Matcher) adjustForInvalidQty(item NormalizedItem, base internal.MatchResult) internal.MatchResult {
	if item.Qty != nil && *item.Qty > 0 {
		return base
	}
	base.Status = internal.MatchReview
	if base.Confidence > 0.7 {
		base.Confidence = 0.7
	}
	return base
}

func (m *Matcher) rankCandidates(query string) []internal.MatchCandidate {
	queryTokens := util.Tokenize(query)
	ids := map[int]struct{}{}

	for _, token := range queryTokens {
		for id := range m.index.TokenToProductIDs[token] {
			ids[id] = struct{}{}
		}
	}

	if len(ids) == 0 {
		i := 0
		for id := range m.index.ProductsByID {
			ids[id] = struct{}{}
			i++
			if i >= 1500 {
				break
			}
		}
	}

	out := make([]internal.MatchCandidate, 0, len(ids))
	for id := range ids {
		product := m.index.ProductsByID[id]
		candidateHeader := m.index.NormalizedHeaderByID[id]
		score := scoreHeader(query, candidateHeader, queryTokens, util.Tokenize(candidateHeader))
		out = append(out, internal.MatchCandidate{ID: product.ID, SyncUID: product.SyncUID, Header: product.Header, Score: score})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func scoreHeader(query, candidate string, queryTokens, candidateTokens []string) float64 {
	dice := util.DiceCoefficient(query, candidate)
	if len(queryTokens) == 0 || len(candidateTokens) == 0 {
		return dice
	}

	set := map[string]struct{}{}
	for _, t := range candidateTokens {
		set[t] = struct{}{}
	}
	overlap := 0
	for _, t := range queryTokens {
		if _, ok := set[t]; ok {
			overlap++
		}
	}
	tokenScore := float64(overlap) / float64(len(queryTokens))
	return 0.65*dice + 0.35*tokenScore
}

func toMatchProduct(p internal.ProductRecord) *internal.MatchProduct {
	id := p.ID
	header := p.Header
	return &internal.MatchProduct{
		ID:         &id,
		SyncUID:    p.SyncUID,
		Header:     &header,
		Articul:    p.Articul,
		UnitHeader: p.UnitHeader,
		FlatCodes:  p.FlatCodes,
	}
}

func toCandidates(products []internal.ProductRecord, score float64) []internal.MatchCandidate {
	limit := len(products)
	if limit > 5 {
		limit = 5
	}
	out := make([]internal.MatchCandidate, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, internal.MatchCandidate{ID: products[i].ID, SyncUID: products[i].SyncUID, Header: products[i].Header, Score: score})
	}
	return out
}
