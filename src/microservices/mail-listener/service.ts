import path from 'node:path';
import { logger } from '../../logger.js';
import { config } from '../../config.js';
import { AppDb } from '../../storage/db.js';
import type { MailProvider } from '../../types.js';
import { GmailConnector } from '../../connectors/gmail/gmailConnector.js';
import { GmailFetchService } from '../../connectors/gmail/fetchService.js';
import { ImapConnector } from '../../connectors/imap/imapConnector.js';
import { ImapFetchService } from '../../connectors/imap/fetchService.js';
import { EmailProcessingService } from '../../pipeline/processEmail.js';
import { exportRowsToXlsx } from '../../pipeline/export/xlsxExporter.js';

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function sanitizeMessageId(messageId: string): string {
  return messageId.replace(/[<>:"/\\|?*\s]+/g, '_').slice(0, 120);
}

export interface ListenerOptions {
  provider: MailProvider;
  label: string;
  fetchMax: number;
  processBatch: number;
  intervalSec: number;
  autoExport: boolean;
  exportDir: string;
}

export class MailListenerService {
  private isRunning = false;

  constructor(
    private readonly db: AppDb,
    private readonly options: ListenerOptions,
  ) {}

  stop(): void {
    this.isRunning = false;
  }

  async runForever(): Promise<void> {
    this.isRunning = true;
    logger.info({ options: this.options }, 'Mail listener started');

    while (this.isRunning) {
      const cycleStart = Date.now();

      try {
        await this.runCycle();
      } catch (error) {
        logger.error({ err: error }, 'Mail listener cycle failed');
      }

      const elapsed = Date.now() - cycleStart;
      const delay = Math.max(0, this.options.intervalSec * 1000 - elapsed);
      if (delay > 0 && this.isRunning) {
        await sleep(delay);
      }
    }

    logger.info('Mail listener stopped');
  }

  private async runCycle(): Promise<void> {
    const fetchResult = await this.fetchMessages();
    const processor = new EmailProcessingService(this.db);

    const queue = this.db
      .listEmailsByStatus('fetched', this.options.processBatch)
      .filter((email) => email.provider === this.options.provider);

    let processedCount = 0;
    for (const email of queue) {
      const processed = await processor.processEmail(email);
      processedCount += 1;

      if (this.options.autoExport && processed.processed > 0) {
        const rows = this.db.getExportRows(email.id);
        if (rows.length) {
          const filename = `${email.id}_${sanitizeMessageId(email.messageId)}.xlsx`;
          const outputPath = path.join(this.options.exportDir, filename);
          await exportRowsToXlsx(rows, outputPath);
          logger.info({ emailId: email.id, outputPath }, 'Listener exported xlsx');
        }
      }
    }

    logger.info(
      {
        provider: this.options.provider,
        fetched: fetchResult.fetched,
        stored: fetchResult.stored,
        processedEmails: processedCount,
      },
      'Mail listener cycle completed',
    );
  }

  private async fetchMessages(): Promise<{ fetched: number; stored: number }> {
    if (this.options.provider === 'gmail') {
      const service = new GmailFetchService(this.db, new GmailConnector());
      return service.fetchAndStore(this.options.label, this.options.fetchMax);
    }

    const service = new ImapFetchService(this.db, new ImapConnector());
    return service.fetchAndStore(this.options.label, this.options.fetchMax);
  }
}

export function defaultListenerOptions(): ListenerOptions {
  return {
    provider: config.mailListenerProvider,
    label: config.mailListenerLabel,
    fetchMax: config.mailListenerFetchMax,
    processBatch: config.mailListenerProcessBatch,
    intervalSec: config.mailListenerIntervalSec,
    autoExport: config.mailListenerAutoExport,
    exportDir: path.join(config.outputDir, 'listener'),
  };
}
