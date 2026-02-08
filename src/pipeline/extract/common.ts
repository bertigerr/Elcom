import type { ExtractionItem, ItemSource } from '../../types.js';
import { parseQty } from '../../utils/qty.js';

const IGNORE_PATTERNS = [
  /^--+$/,
  /^спасибо/i,
  /^с уважением/i,
  /^тел[:\s]/i,
  /^e-?mail[:\s]/i,
  /^http/i,
];

function isLikelyNoise(line: string): boolean {
  if (!line.trim()) {
    return true;
  }
  return IGNORE_PATTERNS.some((pattern) => pattern.test(line.trim()));
}

export function lineToExtractionItem(source: ItemSource, lineNo: number, rawLine: string): ExtractionItem | null {
  const compact = rawLine.replace(/\s+/g, ' ').trim();
  if (!compact || isLikelyNoise(compact)) {
    return null;
  }

  const { qty, unit, qtyRaw } = parseQty(compact);
  let noQty = compact;
  if (qtyRaw) {
    const idx = noQty.lastIndexOf(qtyRaw);
    if (idx >= 0) {
      noQty = `${noQty.slice(0, idx)} ${noQty.slice(idx + qtyRaw.length)}`;
    }
  }

  const name = noQty
    .replace(/\b(шт|штук|pcs|pc|м\.?|метр|kg|кг|уп\.?|компл\.?)\b/gi, ' ')
    .replace(/[;|]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();

  const nameOrCode = name.length > 1 ? name : compact;

  return {
    lineNo,
    source,
    rawLine: compact,
    nameOrCode,
    qty,
    unit,
    meta: {
      qtyRaw,
    },
  };
}

export function dedupeItems(items: ExtractionItem[]): ExtractionItem[] {
  const seen = new Set<string>();
  const out: ExtractionItem[] = [];
  for (const item of items) {
    const key = `${item.source}|${item.rawLine}|${item.qty ?? 'null'}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    out.push(item);
  }
  return out;
}
