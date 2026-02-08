import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { describe, expect, it } from 'vitest';
import { AppDb } from '../../src/storage/db.js';
import { EmailProcessingService } from '../../src/pipeline/processEmail.js';
import { exportRowsToXlsx } from '../../src/pipeline/export/xlsxExporter.js';
import type { ProductRecord } from '../../src/types.js';

describe('smoke email -> items -> matches -> xlsx', () => {
  it('processes fixture email and exports xlsx', async () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'elcom-smoke-'));
    const dbPath = path.join(tempDir, 'app.db');
    const outPath = path.join(tempDir, 'result.xlsx');
    const rawPath = path.join(tempDir, 'fixture.eml');

    fs.copyFileSync(path.join(process.cwd(), 'tests/fixtures/sample_quote.eml'), rawPath);

    const db = new AppDb(dbPath);
    const products: ProductRecord[] = [
      {
        id: 100,
        syncUid: 'sync-100',
        header: 'Кабель ВВГнг 3x2.5',
        articul: 'ELC100',
        unitHeader: 'шт',
        manufacturerHeader: 'Элком',
        multiplicityOrder: 1,
        analogCodes: [],
        flatCodes: {},
        updatedAt: null,
        raw: {},
      },
      {
        id: 101,
        syncUid: 'sync-101',
        header: 'Провод ПВС 2x1.5',
        articul: 'ELC101',
        unitHeader: 'м',
        manufacturerHeader: 'Элком',
        multiplicityOrder: 1,
        analogCodes: [],
        flatCodes: {},
        updatedAt: null,
        raw: {},
      },
    ];

    db.upsertProducts(products);

    const email = db.upsertEmail({
      provider: 'gmail',
      messageId: '<fixture-1@example.com>',
      subject: 'Заявка',
      sender: 'customer@example.com',
      receivedAt: new Date().toISOString(),
      hash: 'fixture-hash',
      rawRef: rawPath,
      status: 'fetched',
    });

    const service = new EmailProcessingService(db);
    const processResult = await service.processEmail(email);

    expect(processResult.processed).toBeGreaterThan(0);

    const rows = db.getExportRows(email.id);
    expect(rows.length).toBeGreaterThan(0);

    await exportRowsToXlsx(rows, outPath);
    expect(fs.existsSync(outPath)).toBe(true);

    db.close();
  });
});
