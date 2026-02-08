import pdfParse from 'pdf-parse';
import type { ExtractionItem } from '../../types.js';
import { lineToExtractionItem } from './common.js';

export async function parsePdf(buffer: Buffer): Promise<ExtractionItem[]> {
  const parsed = await pdfParse(buffer);
  const lines = parsed.text
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);

  const out: ExtractionItem[] = [];
  let lineNo = 0;
  for (const line of lines) {
    lineNo += 1;
    const item = lineToExtractionItem('pdf', lineNo, line);
    if (!item) {
      continue;
    }
    if (!item.nameOrCode || item.qty === null) {
      continue;
    }
    out.push(item);
  }

  return out;
}
