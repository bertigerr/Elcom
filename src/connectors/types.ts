import type { GmailFetchedMessage } from '../types.js';

export interface MailConnector {
  fetchInbox(options: { label: string; max: number }): Promise<GmailFetchedMessage[]>;
}
