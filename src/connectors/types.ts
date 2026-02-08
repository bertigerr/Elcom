import type { FetchedMailMessage } from '../types.js';

export interface MailConnector {
  fetchInbox(options: { label: string; max: number }): Promise<FetchedMailMessage[]>;
}
