import { config } from '../../config.js';
import { logger } from '../../logger.js';
import { AppDb } from '../../storage/db.js';
import { defaultListenerOptions, MailListenerService } from './service.js';

const db = new AppDb(config.dbPath);
const listener = new MailListenerService(db, defaultListenerOptions());
let stopping = false;

const shutdown = (signal: string): void => {
  if (stopping) {
    return;
  }
  stopping = true;
  logger.info({ signal }, 'Shutdown signal received');
  listener.stop();
};

process.on('SIGINT', () => shutdown('SIGINT'));
process.on('SIGTERM', () => shutdown('SIGTERM'));

listener
  .runForever()
  .catch((error) => {
    logger.error({ err: error }, 'Mail listener crashed');
    process.exitCode = 1;
  })
  .finally(() => {
    db.close();
  });
