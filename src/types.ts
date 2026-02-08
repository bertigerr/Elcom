export type ItemSource = 'email_text' | 'email_html_table' | 'xlsx' | 'pdf';

export interface ExtractionItem {
  lineNo: number;
  source: ItemSource;
  rawLine: string;
  nameOrCode: string | null;
  qty: number | null;
  unit: string | null;
  meta?: Record<string, unknown>;
}

export type MatchStatus = 'OK' | 'REVIEW' | 'NOT_FOUND';
export type MatchReason = 'CODE' | 'HEADER' | 'FUZZY' | 'NONE';

export interface ProductFlatCodes {
  elcom?: string;
  manufacturer?: string;
  raec?: string;
  pc?: string;
  etm?: string;
}

export interface ProductRecord {
  id: number;
  syncUid: string | null;
  header: string;
  articul: string | null;
  unitHeader: string | null;
  manufacturerHeader: string | null;
  multiplicityOrder: number | null;
  analogCodes: string[];
  flatCodes: ProductFlatCodes;
  updatedAt: string | null;
  raw: Record<string, unknown>;
}

export interface MatchCandidate {
  id: number;
  syncUid: string | null;
  header: string;
  score: number;
}

export interface MatchProduct {
  id: number | null;
  syncUid: string | null;
  header: string | null;
  articul: string | null;
  unitHeader: string | null;
  flatCodes: ProductFlatCodes;
}

export interface MatchResult {
  status: MatchStatus;
  confidence: number;
  reason: MatchReason;
  product: MatchProduct | null;
  candidates: MatchCandidate[];
}

export interface ProcessedLine {
  item: ExtractionItem;
  match: MatchResult;
}

export interface DetectResult {
  isQuote: boolean;
  score: number;
  reason: string;
}

export type MailProvider = 'gmail' | 'imap';

export interface FetchedMailMessage {
  provider: MailProvider;
  messageId: string;
  subject: string;
  from: string;
  receivedAt: string;
  raw: Buffer;
}

export type GmailFetchedMessage = FetchedMailMessage & { provider: 'gmail' };
export type ImapFetchedMessage = FetchedMailMessage & { provider: 'imap' };

export type IncrementalMode = 'hour_price' | 'hour_stock' | 'day';
