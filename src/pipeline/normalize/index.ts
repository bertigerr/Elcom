import type { ExtractionItem } from '../../types.js';
import { normalizeHeader } from '../../utils/text.js';

export interface NormalizedItem extends ExtractionItem {
  normalizedNameOrCode: string;
}

export function normalizeItems(items: ExtractionItem[]): NormalizedItem[] {
  return items.map((item) => ({
    ...item,
    normalizedNameOrCode: normalizeHeader(item.nameOrCode ?? item.rawLine),
  }));
}
