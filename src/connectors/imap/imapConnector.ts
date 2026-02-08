import { ImapFlow } from 'imapflow';
import type { GmailFetchedMessage } from '../../types.js';
import type { MailConnector } from '../types.js';

export interface ImapConfig {
  host: string;
  port: number;
  secure: boolean;
  auth: {
    user: string;
    pass: string;
  };
}

// Skeleton connector reserved for Yandex/any IMAP provider.
export class ImapConnector implements MailConnector {
  private readonly client: ImapFlow;

  constructor(config: ImapConfig) {
    this.client = new ImapFlow(config);
  }

  async fetchInbox(): Promise<GmailFetchedMessage[]> {
    // MVP intentionally keeps Gmail as active provider; IMAP API contract is ready.
    throw new Error('IMAP connector is scaffolded but not enabled in this MVP runtime');
  }
}
