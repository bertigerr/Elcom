package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"elcom/internal"
	"elcom/internal/util"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if _, err := conn.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		_ = conn.Close()
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.init(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return db, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) init() error {
	schema := `
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
`

	_, err := d.conn.Exec(schema)
	return err
}

func (d *DB) UpsertProducts(products []internal.ProductRecord) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
INSERT INTO products (
  id, syncUid, header, articul, unitHeader,
  flat_elcom, flat_manufacturer, flat_raec, flat_pc, flat_etm,
  analogCodes, updatedAt, manufacturerHeader, multiplicityOrder, raw_json, lastSeenAt
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
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
`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range products {
		analogJSON, _ := json.Marshal(p.AnalogCodes)
		if _, err := stmt.Exec(
			p.ID, p.SyncUID, p.Header, p.Articul, p.UnitHeader,
			p.FlatCodes.Elcom, p.FlatCodes.Manufacturer, p.FlatCodes.Raec, p.FlatCodes.PC, p.FlatCodes.Etm,
			string(analogJSON), p.UpdatedAt, p.ManufacturerHeader, p.MultiplicityOrder, p.RawJSON,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) ListProducts() ([]internal.ProductRecord, error) {
	rows, err := d.conn.Query(`
SELECT id, syncUid, header, articul, unitHeader,
       flat_elcom, flat_manufacturer, flat_raec, flat_pc, flat_etm,
       analogCodes, updatedAt, manufacturerHeader, multiplicityOrder, raw_json
FROM products`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []internal.ProductRecord
	for rows.Next() {
		var p internal.ProductRecord
		var analogJSON string
		if err := rows.Scan(
			&p.ID, &p.SyncUID, &p.Header, &p.Articul, &p.UnitHeader,
			&p.FlatCodes.Elcom, &p.FlatCodes.Manufacturer, &p.FlatCodes.Raec, &p.FlatCodes.PC, &p.FlatCodes.Etm,
			&analogJSON, &p.UpdatedAt, &p.ManufacturerHeader, &p.MultiplicityOrder, &p.RawJSON,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(analogJSON), &p.AnalogCodes)
		out = append(out, p)
	}

	return out, rows.Err()
}

func (d *DB) UpsertEmail(provider, messageID, subject, sender, receivedAt, hash, rawRef, status string) (internal.EmailRow, error) {
	_, err := d.conn.Exec(`
INSERT INTO emails (provider, messageId, subject, sender, receivedAt, hash, status, rawRef)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, messageId) DO UPDATE SET
  subject=excluded.subject,
  sender=excluded.sender,
  receivedAt=excluded.receivedAt,
  hash=excluded.hash,
  rawRef=excluded.rawRef,
  updatedAt=CURRENT_TIMESTAMP
`, provider, messageID, subject, sender, receivedAt, hash, status, rawRef)
	if err != nil {
		return internal.EmailRow{}, err
	}

	row, err := d.GetEmailByProviderMessageID(provider, messageID)
	if err != nil {
		return internal.EmailRow{}, err
	}
	if row == nil {
		return internal.EmailRow{}, errors.New("failed to upsert email")
	}
	return *row, nil
}

func (d *DB) GetEmailByProviderMessageID(provider, messageID string) (*internal.EmailRow, error) {
	var row internal.EmailRow
	err := d.conn.QueryRow(`
SELECT id, provider, messageId, subject, sender, receivedAt, hash, status, rawRef
FROM emails WHERE provider = ? AND messageId = ?
`, provider, messageID).Scan(
		&row.ID, &row.Provider, &row.MessageID, &row.Subject, &row.Sender, &row.ReceivedAt, &row.Hash, &row.Status, &row.RawRef,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (d *DB) GetEmailByID(id int) (*internal.EmailRow, error) {
	var row internal.EmailRow
	err := d.conn.QueryRow(`
SELECT id, provider, messageId, subject, sender, receivedAt, hash, status, rawRef
FROM emails WHERE id = ?
`, id).Scan(
		&row.ID, &row.Provider, &row.MessageID, &row.Subject, &row.Sender, &row.ReceivedAt, &row.Hash, &row.Status, &row.RawRef,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (d *DB) ListEmailsByStatus(status string, limit int) ([]internal.EmailRow, error) {
	rows, err := d.conn.Query(`
SELECT id, provider, messageId, subject, sender, receivedAt, hash, status, rawRef
FROM emails WHERE status = ? ORDER BY receivedAt ASC LIMIT ?
`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []internal.EmailRow
	for rows.Next() {
		var row internal.EmailRow
		if err := rows.Scan(&row.ID, &row.Provider, &row.MessageID, &row.Subject, &row.Sender, &row.ReceivedAt, &row.Hash, &row.Status, &row.RawRef); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (d *DB) UpdateEmailStatus(emailID int, status string) error {
	_, err := d.conn.Exec(`UPDATE emails SET status = ?, updatedAt = CURRENT_TIMESTAMP WHERE id = ?`, status, emailID)
	return err
}

func (d *DB) ClearEmailProcessing(emailID int) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(`SELECT id FROM extractions WHERE emailId = ?`, emailID)
	if err != nil {
		return err
	}
	var extractionIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		extractionIDs = append(extractionIDs, id)
	}
	_ = rows.Close()

	for _, id := range extractionIDs {
		if _, err := tx.Exec(`DELETE FROM matches WHERE extractionId = ?`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM extractions WHERE id = ?`, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) InsertExtraction(emailID int, item internal.ExtractionItem) (int64, error) {
	metaJSON, _ := json.Marshal(item.Meta)
	result, err := d.conn.Exec(`
INSERT INTO extractions (emailId, lineNo, source, rawLine, parsedNameOrCode, parsedQty, parsedUnit, parsedJson)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, emailID, item.LineNo, string(item.Source), item.RawLine, item.NameOrCode, item.Qty, item.Unit, string(metaJSON))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *DB) InsertMatch(extractionID int64, result internal.MatchResult) error {
	candidatesJSON, _ := json.Marshal(result.Candidates)
	var productID *int
	var productSyncUID *string
	if result.Product != nil {
		productID = result.Product.ID
		productSyncUID = result.Product.SyncUID
	}

	_, err := d.conn.Exec(`
INSERT INTO matches (extractionId, status, confidence, reason, productId, productSyncUid, candidatesJson)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, extractionID, string(result.Status), result.Confidence, string(result.Reason), productID, productSyncUID, string(candidatesJSON))
	return err
}

func (d *DB) InsertRun(traceID string, emailID int, timings map[string]float64, counts map[string]int) error {
	timingsJSON, _ := json.Marshal(timings)
	countsJSON, _ := json.Marshal(counts)
	_, err := d.conn.Exec(`INSERT INTO runs (traceId, emailId, timingsJson, countsJson) VALUES (?, ?, ?, ?)`, traceID, emailID, string(timingsJSON), string(countsJSON))
	return err
}

func (d *DB) SetMetadata(key, value string) error {
	_, err := d.conn.Exec(`
INSERT INTO metadata (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updatedAt = CURRENT_TIMESTAMP
`, key, value)
	return err
}

func (d *DB) GetMetadata(key string) (*string, error) {
	var value string
	err := d.conn.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (d *DB) GetExportRows(emailID int) ([]internal.MatchExportRow, error) {
	rows, err := d.conn.Query(`
SELECT
  e.lineNo,
  e.source,
  e.rawLine,
  e.parsedNameOrCode,
  e.parsedQty,
  e.parsedUnit,
  m.status,
  m.confidence,
  m.reason,
  p.id,
  p.syncUid,
  p.header,
  p.articul,
  p.unitHeader,
  p.flat_elcom,
  p.flat_manufacturer,
  p.flat_raec,
  p.flat_pc,
  p.flat_etm,
  m.candidatesJson
FROM extractions e
JOIN matches m ON m.extractionId = e.id
LEFT JOIN products p ON p.id = m.productId
WHERE e.emailId = ?
ORDER BY
  CASE m.status WHEN 'OK' THEN 1 WHEN 'REVIEW' THEN 2 ELSE 3 END,
  e.lineNo ASC
`, emailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []internal.MatchExportRow
	for rows.Next() {
		var row internal.MatchExportRow
		var candidatesJSON string
		if err := rows.Scan(
			&row.InputLineNo,
			&row.Source,
			&row.RawLine,
			&row.ParsedNameOrCode,
			&row.ParsedQty,
			&row.ParsedUnit,
			&row.MatchStatus,
			&row.Confidence,
			&row.MatchReason,
			&row.ProductID,
			&row.ProductSyncUID,
			&row.ProductHeader,
			&row.ProductArticul,
			&row.UnitHeader,
			&row.FlatElcom,
			&row.FlatManufacturer,
			&row.FlatRaec,
			&row.FlatPC,
			&row.FlatEtm,
			&candidatesJSON,
		); err != nil {
			return nil, err
		}

		var candidates []internal.MatchCandidate
		_ = json.Unmarshal([]byte(candidatesJSON), &candidates)
		if len(candidates) > 1 {
			row.Candidate2Header = util.StringPtr(candidates[1].Header)
			row.Candidate2Score = util.FloatPtr(candidates[1].Score)
		}
		out = append(out, row)
	}

	return out, rows.Err()
}

func (d *DB) MustEmailByProviderMessageID(provider, messageID string) (internal.EmailRow, error) {
	row, err := d.GetEmailByProviderMessageID(provider, messageID)
	if err != nil {
		return internal.EmailRow{}, err
	}
	if row == nil {
		return internal.EmailRow{}, fmt.Errorf("email not found: provider=%s messageId=%s", provider, messageID)
	}
	return *row, nil
}
