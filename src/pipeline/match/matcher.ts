import { config } from '../../config.js';
import type { MatchCandidate, MatchResult, ProductRecord } from '../../types.js';
import { buildProductIndex, type ProductIndex } from '../../catalog/index.js';
import { diceCoefficient, looksLikeCode, normalizeCode, normalizeHeader, tokenize } from '../../utils/text.js';
import type { NormalizedItem } from '../normalize/index.js';

function toProductPayload(product: ProductRecord) {
  return {
    id: product.id,
    syncUid: product.syncUid,
    header: product.header,
    articul: product.articul,
    unitHeader: product.unitHeader,
    flatCodes: product.flatCodes,
  };
}

function scoreHeader(query: string, candidate: string, queryTokens: string[], candidateTokens: string[]): number {
  const dice = diceCoefficient(query, candidate);
  if (!queryTokens.length || !candidateTokens.length) {
    return dice;
  }

  const candidateSet = new Set(candidateTokens);
  const overlap = queryTokens.filter((token) => candidateSet.has(token)).length;
  const tokenScore = overlap / queryTokens.length;

  return 0.65 * dice + 0.35 * tokenScore;
}

function rankCandidates(index: ProductIndex, query: string): MatchCandidate[] {
  const qTokens = tokenize(query);
  const candidateIds = new Set<number>();

  for (const token of qTokens) {
    const ids = index.tokenToProductIds.get(token);
    if (!ids) {
      continue;
    }
    for (const id of ids) {
      candidateIds.add(id);
    }
  }

  if (!candidateIds.size) {
    for (const id of index.productsById.keys()) {
      candidateIds.add(id);
      if (candidateIds.size >= 1500) {
        break;
      }
    }
  }

  const candidates: MatchCandidate[] = [];
  for (const id of candidateIds) {
    const product = index.productsById.get(id);
    if (!product) {
      continue;
    }

    const normalizedHeader = index.normalizedHeaderById.get(id) ?? normalizeHeader(product.header);
    const score = scoreHeader(query, normalizedHeader, qTokens, tokenize(normalizedHeader));

    candidates.push({
      id: product.id,
      syncUid: product.syncUid,
      header: product.header,
      score,
    });
  }

  return candidates.sort((a, b) => b.score - a.score).slice(0, 5);
}

function pickUnique(products: ProductRecord[]): ProductRecord | null {
  if (products.length === 1) {
    return products[0];
  }
  return null;
}

function invalidQtyReview(base: MatchResult): MatchResult {
  return {
    ...base,
    status: 'REVIEW',
    confidence: Math.min(base.confidence, 0.7),
  };
}

export class Matcher {
  private readonly index: ProductIndex;

  constructor(products: ProductRecord[]) {
    this.index = buildProductIndex(products);
  }

  match(item: NormalizedItem): MatchResult {
    const normalized = item.normalizedNameOrCode || normalizeHeader(item.rawLine);
    const codeCandidate = normalizeCode(item.nameOrCode ?? item.rawLine);

    if (looksLikeCode(item.nameOrCode ?? '') && codeCandidate) {
      const byCode = this.index.byCode.get(codeCandidate) ?? [];
      if (byCode.length === 1) {
        const result: MatchResult = {
          status: 'OK',
          confidence: 0.99,
          reason: 'CODE',
          product: toProductPayload(byCode[0]),
          candidates: byCode.slice(0, 5).map((p) => ({ id: p.id, syncUid: p.syncUid, header: p.header, score: 0.99 })),
        };
        return item.qty == null || item.qty <= 0 ? invalidQtyReview(result) : result;
      }

      if (byCode.length > 1) {
        return {
          status: 'REVIEW',
          confidence: 0.8,
          reason: 'CODE',
          product: null,
          candidates: byCode.slice(0, 5).map((p) => ({ id: p.id, syncUid: p.syncUid, header: p.header, score: 0.8 })),
        };
      }
    }

    const headerMatches = this.index.byHeader.get(normalized) ?? [];
    const exact = pickUnique(headerMatches);
    if (exact) {
      const result: MatchResult = {
        status: 'OK',
        confidence: 0.95,
        reason: 'HEADER',
        product: toProductPayload(exact),
        candidates: [{ id: exact.id, syncUid: exact.syncUid, header: exact.header, score: 0.95 }],
      };
      return item.qty == null || item.qty <= 0 ? invalidQtyReview(result) : result;
    }

    if (headerMatches.length > 1) {
      return {
        status: 'REVIEW',
        confidence: 0.78,
        reason: 'HEADER',
        product: null,
        candidates: headerMatches.slice(0, 5).map((p) => ({ id: p.id, syncUid: p.syncUid, header: p.header, score: 0.78 })),
      };
    }

    const candidates = rankCandidates(this.index, normalized);
    if (!candidates.length) {
      return {
        status: 'NOT_FOUND',
        confidence: 0,
        reason: 'NONE',
        product: null,
        candidates: [],
      };
    }

    const top1 = candidates[0];
    const top2 = candidates[1];
    const gap = top2 ? top1.score - top2.score : top1.score;
    const best = this.index.productsById.get(top1.id)!;

    let base: MatchResult;
    if (top1.score >= config.matchOkThreshold && gap >= config.matchGapThreshold) {
      base = {
        status: 'OK',
        confidence: top1.score,
        reason: 'FUZZY',
        product: toProductPayload(best),
        candidates,
      };
    } else if (top1.score >= config.matchReviewThreshold) {
      base = {
        status: 'REVIEW',
        confidence: top1.score,
        reason: 'FUZZY',
        product: toProductPayload(best),
        candidates,
      };
    } else {
      base = {
        status: 'NOT_FOUND',
        confidence: top1.score,
        reason: 'NONE',
        product: null,
        candidates,
      };
    }

    if (item.qty == null || item.qty <= 0) {
      return invalidQtyReview(base);
    }

    return base;
  }
}
