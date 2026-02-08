import type { DetectResult } from '../../types.js';

const KEYWORDS = ['заявк', 'кп', 'коммерческ', 'прошу', 'нужно', 'кол-во', 'qty', 'счет'];

export function detectQuoteRequest(payload: {
  subject?: string;
  text?: string;
  html?: string;
  attachmentNames?: string[];
}): DetectResult {
  const subject = (payload.subject ?? '').toLowerCase();
  const text = (payload.text ?? '').toLowerCase();
  const html = (payload.html ?? '').toLowerCase();

  let score = 0;

  for (const keyword of KEYWORDS) {
    if (subject.includes(keyword)) {
      score += 0.2;
    }
    if (text.includes(keyword) || html.includes(keyword)) {
      score += 0.1;
    }
  }

  const qtyPatternHits = (text.match(/\b\d+[\d\s.,]*\s*(шт|м|кг|pcs|pc)?\b/g) ?? []).length;
  if (qtyPatternHits >= 2) {
    score += 0.4;
  } else if (qtyPatternHits === 1) {
    score += 0.2;
  }

  const attachmentNames = payload.attachmentNames ?? [];
  if (attachmentNames.some((name) => /\.(xlsx|xls|pdf)$/i.test(name))) {
    score += 0.25;
  }

  if (/<table/i.test(payload.html ?? '')) {
    score += 0.25;
  }

  const bounded = Math.min(1, score);
  const isQuote = bounded >= 0.45;

  return {
    isQuote,
    score: bounded,
    reason: isQuote ? 'rules_positive' : 'rules_negative',
  };
}
