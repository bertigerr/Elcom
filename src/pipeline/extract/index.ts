import path from 'node:path';
import type { Attachment, ParsedMail } from 'mailparser';
import type { ExtractionItem } from '../../types.js';
import { dedupeItems } from './common.js';
import { parseEmailHtmlTable } from './htmlTable.js';
import { parsePdf } from './pdf.js';
import { parseEmailText } from './plainText.js';
import { parseXlsx } from './xlsx.js';

async function parseAttachment(attachment: Attachment): Promise<ExtractionItem[]> {
  const filename = attachment.filename?.toLowerCase() ?? '';
  const contentType = attachment.contentType.toLowerCase();

  if (filename.endsWith('.xlsx') || filename.endsWith('.xls') || contentType.includes('spreadsheet')) {
    return parseXlsx(attachment.content);
  }

  if (filename.endsWith('.pdf') || contentType.includes('pdf')) {
    return parsePdf(attachment.content);
  }

  return [];
}

export async function extractItemsFromEmail(parsed: ParsedMail): Promise<ExtractionItem[]> {
  const all: ExtractionItem[] = [];

  if (parsed.text) {
    all.push(...parseEmailText(parsed.text));
  }

  if (parsed.html && typeof parsed.html === 'string') {
    all.push(...parseEmailHtmlTable(parsed.html));
  }

  for (const attachment of parsed.attachments ?? []) {
    const items = await parseAttachment(attachment);
    all.push(...items.map((item) => ({ ...item, meta: { ...(item.meta ?? {}), attachment: attachment.filename ?? 'unknown' } })));
  }

  const normalized = all.map((item, i) => ({ ...item, lineNo: i + 1 }));
  return dedupeItems(normalized);
}

export async function extractItemsFromInput(type: 'email_text' | 'email_table' | 'xlsx' | 'pdf', input: string): Promise<ExtractionItem[]> {
  if (type === 'email_text') {
    return parseEmailText(input);
  }

  if (type === 'email_table') {
    return parseEmailHtmlTable(input);
  }

  const fs = await import('node:fs/promises');
  const fullPath = path.resolve(input);
  const buffer = await fs.readFile(fullPath);

  if (type === 'xlsx') {
    return parseXlsx(buffer);
  }

  return parsePdf(buffer);
}
