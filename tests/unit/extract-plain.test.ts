import { describe, expect, it } from 'vitest';
import { parseEmailText } from '../../src/pipeline/extract/plainText.js';

describe('parseEmailText', () => {
  it('extracts basic rows', () => {
    const text = `\nВВГнг 3х2.5 100 шт\nКабель NYM 10 м\n`;
    const items = parseEmailText(text);
    expect(items.length).toBe(2);
    expect(items[0].qty).toBe(100);
    expect(items[1].qty).toBe(10);
  });

  it('skips signature rows', () => {
    const text = `С уважением\nИван\nКабель 5 шт`;
    const items = parseEmailText(text);
    expect(items.length).toBe(1);
  });

  it('supports mixed separators', () => {
    const text = `Позиция: Провод ПВС; кол-во 1,5 м`;
    const items = parseEmailText(text);
    expect(items[0].qty).toBe(1.5);
  });

  it('ignores tiny non-product line', () => {
    const text = `ok\nAB\nКабель 2 шт`;
    const items = parseEmailText(text);
    expect(items.length).toBe(1);
  });

  it('handles code-like line', () => {
    const text = `ELC0100203802 3 шт`;
    const items = parseEmailText(text);
    expect(items.length).toBe(1);
    expect(items[0].nameOrCode).toContain('ELC0100203802');
  });
});
