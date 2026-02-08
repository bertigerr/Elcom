import fs from 'node:fs/promises';
import crypto from 'node:crypto';
import { simpleParser } from 'mailparser';
import { logger } from '../logger.js';
import type { ProcessedLine } from '../types.js';
import { AppDb, type EmailRow } from '../storage/db.js';
import { Matcher } from './match/matcher.js';
import { normalizeItems } from './normalize/index.js';
import { detectQuoteRequest } from './detect/detector.js';
import { extractItemsFromEmail } from './extract/index.js';

export class EmailProcessingService {
  constructor(private readonly db: AppDb) {}

  async processByProviderMessageId(provider: string, messageId: string): Promise<{ emailId: number; processed: number }> {
    const email = this.db.getEmailByProviderMessageId(provider, messageId);
    if (!email) {
      throw new Error(`Email not found for provider=${provider} messageId=${messageId}`);
    }
    return this.processEmail(email);
  }

  async processPending(limit: number): Promise<{ processedEmails: number; processedLines: number }> {
    const pending = this.db.listEmailsByStatus('fetched', limit);
    let processedEmails = 0;
    let processedLines = 0;

    for (const email of pending) {
      const result = await this.processEmail(email);
      processedEmails += 1;
      processedLines += result.processed;
    }

    return { processedEmails, processedLines };
  }

  async processEmail(email: EmailRow): Promise<{ emailId: number; processed: number }> {
    const start = Date.now();
    const raw = await fs.readFile(email.rawRef);
    const parsed = await simpleParser(raw);

    const detect = detectQuoteRequest({
      subject: parsed.subject ?? email.subject,
      text: parsed.text ?? '',
      html: typeof parsed.html === 'string' ? parsed.html : '',
      attachmentNames: (parsed.attachments ?? []).map((a: { filename?: string | null }) => a.filename ?? ''),
    });

    this.db.clearEmailProcessing(email.id);

    if (!detect.isQuote) {
      this.db.updateEmailStatus(email.id, 'skipped');
      this.db.insertRun(
        crypto.randomUUID(),
        email.id,
        { totalMs: Date.now() - start },
        { extracted: 0, ok: 0, review: 0, notFound: 0 },
      );
      return { emailId: email.id, processed: 0 };
    }

    const items = await extractItemsFromEmail(parsed);
    const normalized = normalizeItems(items);

    const matcher = new Matcher(this.db.listProducts());
    const lines: ProcessedLine[] = normalized.map((item) => ({
      item,
      match: matcher.match(item),
    }));

    let ok = 0;
    let review = 0;
    let notFound = 0;

    for (const line of lines) {
      const extractionId = this.db.insertExtraction(email.id, line.item);
      this.db.insertMatch(extractionId, line.match);

      if (line.match.status === 'OK') ok += 1;
      if (line.match.status === 'REVIEW') review += 1;
      if (line.match.status === 'NOT_FOUND') notFound += 1;
    }

    this.db.updateEmailStatus(email.id, 'processed');
    this.db.insertRun(
      crypto.randomUUID(),
      email.id,
      { totalMs: Date.now() - start },
      { extracted: lines.length, ok, review, notFound },
    );

    logger.info({ emailId: email.id, lines: lines.length, ok, review, notFound }, 'Email processed');
    return { emailId: email.id, processed: lines.length };
  }
}
