import { describe, expect, it } from 'vitest';
import { parseQty } from '../../src/utils/qty.js';

describe('parseQty', () => {
  it('parses thousand with space', () => {
    expect(parseQty('Кабель 1 000 шт').qty).toBe(1000);
  });

  it('parses decimal comma', () => {
    expect(parseQty('Провод 1,5 м').qty).toBe(1.5);
  });

  it('parses decimal dot', () => {
    expect(parseQty('Провод 1.5 м').qty).toBe(1.5);
  });

  it('parses thousand dot', () => {
    expect(parseQty('Кабель 1.000 шт').qty).toBe(1000);
  });
});
