import type { ExtractionItem } from '../../types.js';
import { lineToExtractionItem } from './common.js';

export function parseEmailText(text: string): ExtractionItem[] {
  const lines = text
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);

  const out: ExtractionItem[] = [];
  let lineNo = 0;

  for (const line of lines) {
    lineNo += 1;

    const candidate = lineToExtractionItem('email_text', lineNo, line);
    if (!candidate) {
      continue;
    }

    const hasProductLikeSignal = /[A-Za-zА-Яа-я]/.test(candidate.rawLine);
    const hasQtySignal = candidate.qty !== null;
    if (!hasProductLikeSignal || (!hasQtySignal && candidate.rawLine.length < 8)) {
      continue;
    }

    out.push(candidate);
  }

  return out;
}
