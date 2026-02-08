#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { Command } from 'commander';
import { config } from '../config.js';
import { logger } from '../logger.js';
import { CatalogSyncService } from '../catalog/sync.js';
import { ElcomClient } from '../catalog/elcomClient.js';
import { AppDb } from '../storage/db.js';
import { GmailConnector } from '../connectors/gmail/gmailConnector.js';
import { GmailFetchService } from '../connectors/gmail/fetchService.js';
import { EmailProcessingService } from '../pipeline/processEmail.js';
import { exportRowsToXlsx } from '../pipeline/export/xlsxExporter.js';
import { extractItemsFromInput } from '../pipeline/extract/index.js';
import { normalizeItems } from '../pipeline/normalize/index.js';
import { Matcher } from '../pipeline/match/matcher.js';

const program = new Command();
program.name('elcom-matcher').description('Mail→Quote Parser & Catalog Matcher MVP').version('0.1.0');

program
  .command('catalog:initial-sync')
  .description('Fetch full catalog from Elcom API /product/scroll and store locally')
  .action(async () => {
    const db = new AppDb(config.dbPath);
    try {
      const service = new CatalogSyncService(db, new ElcomClient());
      const result = await service.initialSync();
      logger.info(result, 'Initial sync done');
    } finally {
      db.close();
    }
  });

program
  .command('catalog:incremental-sync')
  .description('Incremental catalog sync via one filter mode')
  .requiredOption('--mode <mode>', 'hour_price|hour_stock|day')
  .action(async (opts: { mode: 'hour_price' | 'hour_stock' | 'day' }) => {
    const db = new AppDb(config.dbPath);
    try {
      const service = new CatalogSyncService(db, new ElcomClient());
      const result = await service.incrementalSync(opts.mode);
      logger.info(result, 'Incremental sync done');
    } finally {
      db.close();
    }
  });

program
  .command('mail:fetch')
  .description('Fetch messages from mail provider into local storage')
  .option('--provider <provider>', 'mail provider', 'gmail')
  .option('--label <label>', 'mailbox/label', 'INBOX')
  .option('--max <max>', 'max messages', '50')
  .action(async (opts: { provider: string; label: string; max: string }) => {
    if (opts.provider !== 'gmail') {
      throw new Error(`Only gmail provider is active in MVP. Received: ${opts.provider}`);
    }

    const db = new AppDb(config.dbPath);
    try {
      const fetchService = new GmailFetchService(db, new GmailConnector());
      const result = await fetchService.fetchAndStore(opts.label, Number(opts.max));
      logger.info(result, 'Mail fetch done');
    } finally {
      db.close();
    }
  });

program
  .command('mail:process')
  .description('Process a fetched email by messageId or process pending batch')
  .option('--provider <provider>', 'mail provider', 'gmail')
  .option('--messageId <messageId>', 'specific messageId to process')
  .option('--batch <batch>', 'batch size for pending fetched emails', '20')
  .action(async (opts: { provider: string; messageId?: string; batch: string }) => {
    const db = new AppDb(config.dbPath);
    try {
      const service = new EmailProcessingService(db);
      if (opts.messageId) {
        const result = await service.processByProviderMessageId(opts.provider, opts.messageId);
        logger.info(result, 'Processed single email');
        return;
      }

      const result = await service.processPending(Number(opts.batch));
      logger.info(result, 'Processed pending emails');
    } finally {
      db.close();
    }
  });

program
  .command('export:xlsx')
  .description('Export processed lines to xlsx')
  .requiredOption('--emailId <emailId>', 'internal email id')
  .requiredOption('--out <out>', 'output xlsx path')
  .action(async (opts: { emailId: string; out: string }) => {
    const db = new AppDb(config.dbPath);
    try {
      const emailId = Number(opts.emailId);
      const rows = db.getExportRows(emailId);
      if (!rows.length) {
        throw new Error(`No export rows for emailId=${emailId}`);
      }
      await exportRowsToXlsx(rows, path.resolve(opts.out));
      logger.info({ out: opts.out, rows: rows.length }, 'Export completed');
    } finally {
      db.close();
    }
  });

program
  .command('run')
  .description('One-off parse+match from input text/file')
  .requiredOption('--input <input>', 'input file path or raw text')
  .requiredOption('--type <type>', 'xlsx|pdf|email_text|email_table')
  .requiredOption('--output <output>', 'output xlsx file')
  .option('--use-cache <cache>', 'not used directly in sqlite MVP, kept for compatibility')
  .action(async (opts: { input: string; type: 'xlsx' | 'pdf' | 'email_text' | 'email_table'; output: string }) => {
    const db = new AppDb(config.dbPath);
    try {
      const inputValue = fs.existsSync(opts.input) ? opts.input : opts.input;
      const items = await extractItemsFromInput(opts.type, inputValue);
      const matcher = new Matcher(db.listProducts());
      const rows = normalizeItems(items).map((item) => {
        const match = matcher.match(item);
        return {
          input_line_no: item.lineNo,
          source: item.source,
          raw_line: item.rawLine,
          parsed_name_or_code: item.nameOrCode,
          parsed_qty: item.qty,
          parsed_unit: item.unit,
          match_status: match.status,
          confidence: match.confidence,
          match_reason: match.reason,
          product_id: match.product?.id ?? null,
          product_syncUid: match.product?.syncUid ?? null,
          product_header: match.product?.header ?? null,
          product_articul: match.product?.articul ?? null,
          unitHeader: match.product?.unitHeader ?? null,
          flat_elcom: match.product?.flatCodes.elcom ?? null,
          flat_manufacturer: match.product?.flatCodes.manufacturer ?? null,
          flat_raec: match.product?.flatCodes.raec ?? null,
          flat_pc: match.product?.flatCodes.pc ?? null,
          flat_etm: match.product?.flatCodes.etm ?? null,
          candidate2_header: match.candidates[1]?.header ?? null,
          candidate2_score: match.candidates[1]?.score ?? null,
        };
      });

      await exportRowsToXlsx(rows, path.resolve(opts.output));
      logger.info({ output: opts.output, rows: rows.length }, 'Run completed');
    } finally {
      db.close();
    }
  });

program.parseAsync().catch((error) => {
  logger.error({ err: error }, 'CLI failed');
  process.exitCode = 1;
});
