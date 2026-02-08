import { ImapFlow, type SearchObject } from 'imapflow';
import { config, requireEnv } from '../../config.js';
import type { ImapFetchedMessage } from '../../types.js';
import type { MailConnector } from '../types.js';

export interface ImapConfig {
  host: string;
  port: number;
  secure: boolean;
  auth: {
    user: string;
    pass: string;
  };
  markSeen?: boolean;
}

function toIsoDate(value: Date | string | undefined): string {
  if (!value) {
    return new Date().toISOString();
  }
  if (value instanceof Date) {
    return value.toISOString();
  }
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? new Date().toISOString() : parsed.toISOString();
}

function getHeaderValue(headers: Map<string, string> | undefined, key: string): string {
  if (!headers) {
    return '';
  }
  return headers.get(key.toLowerCase()) ?? headers.get(key) ?? '';
}

export class ImapConnector implements MailConnector {
  private readonly client: ImapFlow;
  private readonly markSeen: boolean;

  constructor(imapConfig?: ImapConfig) {
    const cfg: ImapConfig =
      imapConfig ?? {
        host: requireEnv(config.imapHost, 'IMAP_HOST'),
        port: config.imapPort,
        secure: config.imapSecure,
        auth: {
          user: requireEnv(config.imapUser, 'IMAP_USER'),
          pass: requireEnv(config.imapPassword, 'IMAP_PASSWORD'),
        },
        markSeen: config.imapMarkSeen,
      };

    this.client = new ImapFlow({
      host: cfg.host,
      port: cfg.port,
      secure: cfg.secure,
      auth: cfg.auth,
      logger: false,
    });

    this.markSeen = Boolean(cfg.markSeen);
  }

  async fetchInbox(options: { label: string; max: number }): Promise<ImapFetchedMessage[]> {
    await this.client.connect();
    const lock = await this.client.getMailboxLock(options.label);

    try {
      const criteria: SearchObject = { seen: false };
      const uids = await this.client.search(criteria, { uid: true });
      const selected = (Array.isArray(uids) ? uids : []).slice(-options.max);
      if (!selected.length) {
        return [];
      }

      const out: ImapFetchedMessage[] = [];
      for await (const msg of this.client.fetch(
        selected,
        { uid: true, envelope: true, source: true, internalDate: true, headers: true },
        { uid: true },
      )) {
        const source = msg.source;
        if (!source) {
          continue;
        }

        const sourceBuffer = Buffer.isBuffer(source) ? source : Buffer.from(source as Uint8Array);
        const headerMap = msg.headers as Map<string, string> | undefined;
        const messageId = msg.envelope?.messageId ?? getHeaderValue(headerMap, 'message-id') ?? String(msg.uid);

        out.push({
          provider: 'imap',
          messageId,
          subject: msg.envelope?.subject ?? getHeaderValue(headerMap, 'subject') ?? '',
          from:
            msg.envelope?.from?.map((f) => `${f.name ?? ''} <${f.address ?? ''}>`).join(', ') ??
            getHeaderValue(headerMap, 'from') ??
            '',
          receivedAt: toIsoDate(msg.internalDate),
          raw: sourceBuffer,
        });

        if (this.markSeen && msg.uid) {
          await this.client.messageFlagsAdd(msg.uid, ['\\Seen'], { uid: true });
        }
      }

      return out;
    } finally {
      lock.release();
      await this.client.logout();
    }
  }
}
