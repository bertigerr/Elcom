import { google } from 'googleapis';
import { config, requireEnv } from '../../config.js';
import { logger } from '../../logger.js';
import type { GmailFetchedMessage } from '../../types.js';
import type { MailConnector } from '../types.js';

function decodeBase64Url(input: string): Buffer {
  const normalized = input.replace(/-/g, '+').replace(/_/g, '/');
  const padding = normalized.length % 4 === 0 ? '' : '='.repeat(4 - (normalized.length % 4));
  return Buffer.from(`${normalized}${padding}`, 'base64');
}

export class GmailConnector implements MailConnector {
  private readonly gmail = google.gmail({
    version: 'v1',
    auth: (() => {
      const client = new google.auth.OAuth2(
        requireEnv(config.gmailClientId, 'GMAIL_CLIENT_ID'),
        requireEnv(config.gmailClientSecret, 'GMAIL_CLIENT_SECRET'),
        config.gmailRedirectUri,
      );
      client.setCredentials({
        refresh_token: requireEnv(config.gmailRefreshToken, 'GMAIL_REFRESH_TOKEN'),
      });
      return client;
    })(),
  });

  async fetchInbox(options: { label: string; max: number }): Promise<GmailFetchedMessage[]> {
    const list = await this.gmail.users.messages.list({
      userId: 'me',
      maxResults: options.max,
      labelIds: [options.label],
    });

    const messages = list.data.messages ?? [];
    const out: GmailFetchedMessage[] = [];

    for (const msg of messages) {
      if (!msg.id) {
        continue;
      }

      const rawResponse = await this.gmail.users.messages.get({
        userId: 'me',
        id: msg.id,
        format: 'raw',
      });

      const metaResponse = await this.gmail.users.messages.get({
        userId: 'me',
        id: msg.id,
        format: 'metadata',
        metadataHeaders: ['Subject', 'From', 'Date', 'Message-ID'],
      });

      const headers = metaResponse.data.payload?.headers ?? [];
      const subject = headers.find((h) => h.name?.toLowerCase() === 'subject')?.value ?? '';
      const from = headers.find((h) => h.name?.toLowerCase() === 'from')?.value ?? '';
      const date = headers.find((h) => h.name?.toLowerCase() === 'date')?.value;
      const messageId = headers.find((h) => h.name?.toLowerCase() === 'message-id')?.value ?? msg.id;

      const raw = rawResponse.data.raw;
      if (!raw) {
        logger.warn({ id: msg.id }, 'Skipping Gmail message without raw payload');
        continue;
      }

      out.push({
        provider: 'gmail',
        messageId,
        subject,
        from,
        receivedAt: date ? new Date(date).toISOString() : new Date().toISOString(),
        raw: decodeBase64Url(raw),
      });
    }

    return out;
  }
}
