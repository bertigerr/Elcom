import { logger } from '../../logger.js';
import { AppDb } from '../../storage/db.js';
import { MailStoreService } from '../mailStoreService.js';
import { ImapConnector } from './imapConnector.js';

export class ImapFetchService {
  private readonly storeService: MailStoreService;

  constructor(
    private readonly db: AppDb,
    private readonly connector: ImapConnector,
  ) {
    this.storeService = new MailStoreService(db);
  }

  async fetchAndStore(label: string, max: number): Promise<{ fetched: number; stored: number }> {
    const fetched = await this.connector.fetchInbox({ label, max });

    let stored = 0;
    for (const msg of fetched) {
      this.storeService.store(msg);
      stored += 1;
    }

    logger.info({ fetched: fetched.length, stored }, 'IMAP fetch completed');
    return { fetched: fetched.length, stored };
  }
}
