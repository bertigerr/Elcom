import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { config } from '../config.js';
import { AppDb } from '../storage/db.js';
import type { FetchedMailMessage } from '../types.js';

function hashRaw(raw: Buffer): string {
  return crypto.createHash('sha256').update(raw).digest('hex');
}

function buildRawPath(messageHash: string): string {
  return path.join(config.rawMailDir, `${messageHash}.eml`);
}

export class MailStoreService {
  constructor(private readonly db: AppDb) {}

  store(message: FetchedMailMessage): void {
    const hash = hashRaw(message.raw);
    const rawPath = buildRawPath(hash);

    fs.mkdirSync(config.rawMailDir, { recursive: true });
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
  }
}
