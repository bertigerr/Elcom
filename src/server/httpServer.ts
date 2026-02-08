import http from 'node:http';
import path from 'node:path';
import { AppDb } from '../storage/db.js';
import { config } from '../config.js';
import { EmailProcessingService } from '../pipeline/processEmail.js';
import { exportRowsToXlsx } from '../pipeline/export/xlsxExporter.js';

export function createServer() {
  const db = new AppDb(config.dbPath);
  const processor = new EmailProcessingService(db);

  return http.createServer(async (req, res) => {
    try {
      if (!req.url || !req.method) {
        res.writeHead(400).end('Bad request');
        return;
      }

      if (req.method === 'POST' && req.url.startsWith('/process/email/')) {
        const messageId = decodeURIComponent(req.url.replace('/process/email/', ''));
        const result = await processor.processByProviderMessageId('gmail', messageId);
        res.writeHead(200, { 'content-type': 'application/json' }).end(JSON.stringify(result));
        return;
      }

      if (req.method === 'GET' && req.url.startsWith('/export/email/')) {
        const emailIdRaw = req.url.replace('/export/email/', '').replace('.xlsx', '');
        const emailId = Number(emailIdRaw);
        const rows = db.getExportRows(emailId);
        const tempPath = path.join(config.outputDir, `email_${emailId}.xlsx`);
        await exportRowsToXlsx(rows, tempPath);
        res.writeHead(200, { 'content-type': 'application/json' }).end(JSON.stringify({ path: tempPath }));
        return;
      }

      res.writeHead(404).end('Not found');
    } catch (error) {
      res.writeHead(500).end(String(error));
    }
  });
}
