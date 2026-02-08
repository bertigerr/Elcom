import fs from 'node:fs';
import path from 'node:path';
import Database from 'better-sqlite3';
import type { ExtractionItem, MatchResult, ProductFlatCodes, ProductRecord } from '../types.js';

export interface EmailRow {
  id: number;
  provider: string;
  messageId: string;
  subject: string;
  sender: string;
  receivedAt: string;
  hash: string;
  status: string;
  rawRef: string;
}

export interface MatchExportRow {
  input_line_no: number;
  source: string;
  raw_line: string;
  parsed_name_or_code: string | null;
  parsed_qty: number | null;
  parsed_unit: string | null;
  match_status: string;
  confidence: number;
  match_reason: string;
  product_id: number | null;
  product_syncUid: string | null;
  product_header: string | null;
  product_articul: string | null;
  unitHeader: string | null;
  flat_elcom: string | null;
  flat_manufacturer: string | null;
  flat_raec: string | null;
  flat_pc: string | null;
  flat_etm: string | null;
  candidate2_header: string | null;
  candidate2_score: number | null;
}

function safeParseJson<T>(value: string | null): T {
  if (!value) {
    return {} as T;
  }
  try {
    return JSON.parse(value) as T;
  } catch {
    return {} as T;
  }
}

export class AppDb {
  readonly db: Database.Database;

  constructor(dbPath: string) {
    fs.mkdirSync(path.dirname(dbPath), { recursive: true });
    this.db = new Database(dbPath);
    this.db.pragma('journal_mode = WAL');
    this.init();
  }

  close(): void {
    this.db.close();
  }

  private init(): void {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS products (
        id INTEGER PRIMARY KEY,
        syncUid TEXT,
        header TEXT NOT NULL,
        articul TEXT,
        unitHeader TEXT,
        flat_elcom TEXT,
        flat_manufacturer TEXT,
        flat_raec TEXT,
        flat_pc TEXT,
        flat_etm TEXT,
        analogCodes TEXT,
        updatedAt TEXT,
        manufacturerHeader TEXT,
        multiplicityOrder REAL,
        raw_json TEXT NOT NULL,
        lastSeenAt TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
      );

      CREATE INDEX IF NOT EXISTS idx_products_header ON products(header);
      CREATE INDEX IF NOT EXISTS idx_products_articul ON products(articul);
      CREATE INDEX IF NOT EXISTS idx_products_syncUid ON products(syncUid);

      CREATE TABLE IF NOT EXISTS emails (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        provider TEXT NOT NULL,
        messageId TEXT NOT NULL,
        subject TEXT,
        sender TEXT,
        receivedAt TEXT,
        hash TEXT NOT NULL,
        status TEXT NOT NULL DEFAULT 'fetched',
        rawRef TEXT NOT NULL,
        createdAt TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updatedAt TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(provider, messageId)
      );

      CREATE TABLE IF NOT EXISTS extractions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        emailId INTEGER NOT NULL,
        lineNo INTEGER NOT NULL,
        source TEXT NOT NULL,
        rawLine TEXT NOT NULL,
        parsedNameOrCode TEXT,
        parsedQty REAL,
        parsedUnit TEXT,
        parsedJson TEXT NOT NULL,
        createdAt TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(emailId, lineNo, source, rawLine),
        FOREIGN KEY(emailId) REFERENCES emails(id)
      );

      CREATE TABLE IF NOT EXISTS matches (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        extractionId INTEGER NOT NULL UNIQUE,
        status TEXT NOT NULL,
        confidence REAL NOT NULL,
        reason TEXT NOT NULL,
        productId INTEGER,
        productSyncUid TEXT,
        candidatesJson TEXT NOT NULL,
        createdAt TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(extractionId) REFERENCES extractions(id)
      );

      CREATE TABLE IF NOT EXISTS runs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        traceId TEXT NOT NULL,
        emailId INTEGER,
        timingsJson TEXT NOT NULL,
        countsJson TEXT NOT NULL,
        createdAt TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(emailId) REFERENCES emails(id)
      );

      CREATE TABLE IF NOT EXISTS metadata (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL,
        updatedAt TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
      );
    `);
  }

  upsertProducts(products: ProductRecord[]): void {
    const stmt = this.db.prepare(`
      INSERT INTO products (
        id, syncUid, header, articul, unitHeader,
        flat_elcom, flat_manufacturer, flat_raec, flat_pc, flat_etm,
        analogCodes, updatedAt, manufacturerHeader, multiplicityOrder, raw_json, lastSeenAt
      ) VALUES (
        @id, @syncUid, @header, @articul, @unitHeader,
        @flat_elcom, @flat_manufacturer, @flat_raec, @flat_pc, @flat_etm,
        @analogCodes, @updatedAt, @manufacturerHeader, @multiplicityOrder, @raw_json, CURRENT_TIMESTAMP
      )
      ON CONFLICT(id) DO UPDATE SET
        syncUid=excluded.syncUid,
        header=excluded.header,
        articul=excluded.articul,
        unitHeader=excluded.unitHeader,
        flat_elcom=excluded.flat_elcom,
        flat_manufacturer=excluded.flat_manufacturer,
        flat_raec=excluded.flat_raec,
        flat_pc=excluded.flat_pc,
        flat_etm=excluded.flat_etm,
        analogCodes=excluded.analogCodes,
        updatedAt=excluded.updatedAt,
        manufacturerHeader=excluded.manufacturerHeader,
        multiplicityOrder=excluded.multiplicityOrder,
        raw_json=excluded.raw_json,
        lastSeenAt=CURRENT_TIMESTAMP
    `);

    const trx = this.db.transaction((rows: ProductRecord[]) => {
      for (const p of rows) {
        const flat = p.flatCodes ?? {};
        stmt.run({
          id: p.id,
          syncUid: p.syncUid,
          header: p.header,
          articul: p.articul,
          unitHeader: p.unitHeader,
          flat_elcom: flat.elcom ?? null,
          flat_manufacturer: flat.manufacturer ?? null,
          flat_raec: flat.raec ?? null,
          flat_pc: flat.pc ?? null,
          flat_etm: flat.etm ?? null,
          analogCodes: JSON.stringify(p.analogCodes ?? []),
          updatedAt: p.updatedAt,
          manufacturerHeader: p.manufacturerHeader,
          multiplicityOrder: p.multiplicityOrder,
          raw_json: JSON.stringify(p.raw ?? {}),
        });
      }
    });

    trx(products);
  }

  listProducts(): ProductRecord[] {
    const rows = this.db.prepare('SELECT * FROM products').all() as Record<string, unknown>[];
    return rows.map((r) => ({
      id: Number(r.id),
      syncUid: (r.syncUid as string | null) ?? null,
      header: String(r.header ?? ''),
      articul: (r.articul as string | null) ?? null,
      unitHeader: (r.unitHeader as string | null) ?? null,
      manufacturerHeader: (r.manufacturerHeader as string | null) ?? null,
      multiplicityOrder: r.multiplicityOrder == null ? null : Number(r.multiplicityOrder),
      analogCodes: safeParseJson<string[]>((r.analogCodes as string | null) ?? null) ?? [],
      flatCodes: {
        elcom: (r.flat_elcom as string | null) ?? undefined,
        manufacturer: (r.flat_manufacturer as string | null) ?? undefined,
        raec: (r.flat_raec as string | null) ?? undefined,
        pc: (r.flat_pc as string | null) ?? undefined,
        etm: (r.flat_etm as string | null) ?? undefined,
      },
      updatedAt: (r.updatedAt as string | null) ?? null,
      raw: safeParseJson<Record<string, unknown>>((r.raw_json as string | null) ?? null),
    }));
  }

  getEmailByProviderMessageId(provider: string, messageId: string): EmailRow | null {
    const row = this.db
      .prepare('SELECT * FROM emails WHERE provider = ? AND messageId = ?')
      .get(provider, messageId) as EmailRow | undefined;
    return row ?? null;
  }

  getEmailById(id: number): EmailRow | null {
    const row = this.db.prepare('SELECT * FROM emails WHERE id = ?').get(id) as EmailRow | undefined;
    return row ?? null;
  }

  listEmailsByStatus(status: string, limit = 50): EmailRow[] {
    return this.db
      .prepare('SELECT * FROM emails WHERE status = ? ORDER BY receivedAt ASC LIMIT ?')
      .all(status, limit) as EmailRow[];
  }

  upsertEmail(payload: Omit<EmailRow, 'id' | 'status'> & { status?: string }): EmailRow {
    const status = payload.status ?? 'fetched';
    this.db
      .prepare(`
        INSERT INTO emails (provider, messageId, subject, sender, receivedAt, hash, status, rawRef)
        VALUES (@provider, @messageId, @subject, @sender, @receivedAt, @hash, @status, @rawRef)
        ON CONFLICT(provider, messageId) DO UPDATE SET
          subject=excluded.subject,
          sender=excluded.sender,
          receivedAt=excluded.receivedAt,
          hash=excluded.hash,
          rawRef=excluded.rawRef,
          updatedAt=CURRENT_TIMESTAMP
      `)
      .run({ ...payload, status });

    const row = this.getEmailByProviderMessageId(payload.provider, payload.messageId);
    if (!row) {
      throw new Error('Failed to upsert email');
    }
    return row;
  }

  updateEmailStatus(emailId: number, status: string): void {
    this.db
      .prepare('UPDATE emails SET status = ?, updatedAt = CURRENT_TIMESTAMP WHERE id = ?')
      .run(status, emailId);
  }

  clearEmailProcessing(emailId: number): void {
    const extractionRows = this.db
      .prepare('SELECT id FROM extractions WHERE emailId = ?')
      .all(emailId) as Array<{ id: number }>;

    const deleteMatch = this.db.prepare('DELETE FROM matches WHERE extractionId = ?');
    const deleteExtraction = this.db.prepare('DELETE FROM extractions WHERE id = ?');

    const trx = this.db.transaction(() => {
      for (const row of extractionRows) {
        deleteMatch.run(row.id);
        deleteExtraction.run(row.id);
      }
    });

    trx();
  }

  insertExtraction(emailId: number, item: ExtractionItem): number {
    const result = this.db
      .prepare(`
        INSERT INTO extractions (emailId, lineNo, source, rawLine, parsedNameOrCode, parsedQty, parsedUnit, parsedJson)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
      `)
      .run(
        emailId,
        item.lineNo,
        item.source,
        item.rawLine,
        item.nameOrCode,
        item.qty,
        item.unit,
        JSON.stringify(item.meta ?? {}),
      );

    return Number(result.lastInsertRowid);
  }

  insertMatch(extractionId: number, result: MatchResult): void {
    this.db
      .prepare(`
        INSERT INTO matches (extractionId, status, confidence, reason, productId, productSyncUid, candidatesJson)
        VALUES (?, ?, ?, ?, ?, ?, ?)
      `)
      .run(
        extractionId,
        result.status,
        result.confidence,
        result.reason,
        result.product?.id ?? null,
        result.product?.syncUid ?? null,
        JSON.stringify(result.candidates ?? []),
      );
  }

  insertRun(traceId: string, emailId: number, timings: Record<string, number>, counts: Record<string, number>): void {
    this.db
      .prepare('INSERT INTO runs (traceId, emailId, timingsJson, countsJson) VALUES (?, ?, ?, ?)')
      .run(traceId, emailId, JSON.stringify(timings), JSON.stringify(counts));
  }

  setMetadata(key: string, value: string): void {
    this.db
      .prepare(
        `INSERT INTO metadata (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updatedAt = CURRENT_TIMESTAMP`,
      )
      .run(key, value);
  }

  getMetadata(key: string): string | null {
    const row = this.db.prepare('SELECT value FROM metadata WHERE key = ?').get(key) as
      | { value: string }
      | undefined;
    return row?.value ?? null;
  }

  getExportRows(emailId: number): MatchExportRow[] {
    const rows = this.db
      .prepare(`
        SELECT
          e.lineNo as input_line_no,
          e.source as source,
          e.rawLine as raw_line,
          e.parsedNameOrCode as parsed_name_or_code,
          e.parsedQty as parsed_qty,
          e.parsedUnit as parsed_unit,
          m.status as match_status,
          m.confidence as confidence,
          m.reason as match_reason,
          p.id as product_id,
          p.syncUid as product_syncUid,
          p.header as product_header,
          p.articul as product_articul,
          p.unitHeader as unitHeader,
          p.flat_elcom as flat_elcom,
          p.flat_manufacturer as flat_manufacturer,
          p.flat_raec as flat_raec,
          p.flat_pc as flat_pc,
          p.flat_etm as flat_etm,
          m.candidatesJson as candidatesJson
        FROM extractions e
        JOIN matches m ON m.extractionId = e.id
        LEFT JOIN products p ON p.id = m.productId
        WHERE e.emailId = ?
        ORDER BY
          CASE m.status WHEN 'OK' THEN 1 WHEN 'REVIEW' THEN 2 ELSE 3 END,
          e.lineNo ASC
      `)
      .all(emailId) as Array<Record<string, unknown>>;

    return rows.map((row) => {
      const candidates = safeParseJson<Array<{ header?: string; score?: number }>>(row.candidatesJson as string);
      const c2 = candidates[1];
      return {
        input_line_no: Number(row.input_line_no),
        source: String(row.source),
        raw_line: String(row.raw_line),
        parsed_name_or_code: (row.parsed_name_or_code as string | null) ?? null,
        parsed_qty: row.parsed_qty == null ? null : Number(row.parsed_qty),
        parsed_unit: (row.parsed_unit as string | null) ?? null,
        match_status: String(row.match_status),
        confidence: Number(row.confidence),
        match_reason: String(row.match_reason),
        product_id: row.product_id == null ? null : Number(row.product_id),
        product_syncUid: (row.product_syncUid as string | null) ?? null,
        product_header: (row.product_header as string | null) ?? null,
        product_articul: (row.product_articul as string | null) ?? null,
        unitHeader: (row.unitHeader as string | null) ?? null,
        flat_elcom: (row.flat_elcom as string | null) ?? null,
        flat_manufacturer: (row.flat_manufacturer as string | null) ?? null,
        flat_raec: (row.flat_raec as string | null) ?? null,
        flat_pc: (row.flat_pc as string | null) ?? null,
        flat_etm: (row.flat_etm as string | null) ?? null,
        candidate2_header: c2?.header ?? null,
        candidate2_score: c2?.score ?? null,
      };
    });
  }
}

export function toFlatCodes(raw: Record<string, unknown>): ProductFlatCodes {
  const flat = (raw.flatCodes as Record<string, unknown> | undefined) ?? {};
  return {
    elcom: (flat.elcom as string | undefined) ?? undefined,
    manufacturer: (flat.manufacturer as string | undefined) ?? undefined,
    raec: (flat.raec as string | undefined) ?? undefined,
    pc: (flat.pc as string | undefined) ?? undefined,
    etm: (flat.etm as string | undefined) ?? undefined,
  };
}
