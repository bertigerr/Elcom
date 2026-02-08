import { describe, expect, it } from 'vitest';
import { Matcher } from '../../src/pipeline/match/matcher.js';
import type { ProductRecord } from '../../src/types.js';

const products: ProductRecord[] = [
  {
    id: 1,
    syncUid: 'sync-1',
    header: 'Кабель ВВГнг 3x2.5',
    articul: 'ELC0100203802',
    unitHeader: 'м',
    manufacturerHeader: 'Элком',
    multiplicityOrder: 1,
    analogCodes: [],
    flatCodes: { manufacturer: 'MNF-123' },
    updatedAt: null,
    raw: {},
  },
  {
    id: 2,
    syncUid: 'sync-2',
    header: 'Кабель ВВГнг 3x4',
    articul: 'ELC0100203803',
    unitHeader: 'м',
    manufacturerHeader: 'Элком',
    multiplicityOrder: 1,
    analogCodes: [],
    flatCodes: {},
    updatedAt: null,
    raw: {},
  },
];

const matcher = new Matcher(products);

describe('matcher', () => {
  it('exact by code -> OK', () => {
    const result = matcher.match({
      lineNo: 1,
      source: 'email_text',
      rawLine: 'ELC0100203802 2 шт',
      nameOrCode: 'ELC0100203802',
      qty: 2,
      unit: 'шт',
      normalizedNameOrCode: 'ELC0100203802',
    });

    expect(result.status).toBe('OK');
    expect(result.reason).toBe('CODE');
    expect(result.product?.id).toBe(1);
  });

  it('fuzzy ambiguous -> REVIEW', () => {
    const result = matcher.match({
      lineNo: 2,
      source: 'email_text',
      rawLine: 'Кабель ВВГнг 3х',
      nameOrCode: 'Кабель ВВГнг 3х',
      qty: 3,
      unit: 'м',
      normalizedNameOrCode: 'КАБЕЛЬ ВВГНГ 3X',
    });

    expect(result.status).toBe('REVIEW');
    expect(result.candidates.length).toBeGreaterThan(0);
  });

  it('not found -> NOT_FOUND', () => {
    const result = matcher.match({
      lineNo: 3,
      source: 'email_text',
      rawLine: 'Совсем другой товар 5 шт',
      nameOrCode: 'Совсем другой товар',
      qty: 5,
      unit: 'шт',
      normalizedNameOrCode: 'СОВСЕМ ДРУГОЙ ТОВАР',
    });

    expect(['NOT_FOUND', 'REVIEW']).toContain(result.status);
  });
});
