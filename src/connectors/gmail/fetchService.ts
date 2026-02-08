import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { config } from '../../config.js';
import { logger } from '../../logger.js';
import { AppDb } from '../../storage/db.js';
import type { GmailFetchedMessage } from '../../types.js';
import { GmailConnector } from './gmailConnector.js';

function hashRaw(raw: Buffer): string {
  return crypto.createHash('sha256').update(raw).digest('hex');
}

function buildRawPath(messageHash: string): string {
  return path.join(config.rawMailDir, `${messageHash}.eml`);
}

export class GmailFetchService {
  constructor(
    private readonly db: AppDb,
    private readonly connector: GmailConnector,
  ) {}

  async fetchAndStore(label: string, max: number): Promise<{ fetched: number; stored: number }> {
    const fetched = await this.connector.fetchInbox({ label, max });
    fs.mkdirSync(config.rawMailDir, { recursive: true });

    let stored = 0;
    for (const msg of fetched) {
      const result = this.storeMessage(msg);
      stored += result ? 1 : 0;
    }

    logger.info({ fetched: fetched.length, stored }, 'Gmail fetch completed');
    return { fetched: fetched.length, stored };
  }

  private storeMessage(message: GmailFetchedMessage): boolean {
    const hash = hashRaw(message.raw);
    const rawPath = buildRawPath(hash);

    if (!fs.existsSync(rawPath)) {
      fs.writeFileSync(rawPath, message.raw);
    }

    this.db.upsertEmail({
      provider: message.provider,
      messageId: message.messageId,
      subject: message.subject,
      sender: message.from,
      receivedAt: message.receivedAt,
      hash,
      rawRef: rawPath,
      status: 'fetched',
    });

    return true;
  }
}
