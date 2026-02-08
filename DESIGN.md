# DESIGN: Mail -> Quote Parser & Catalog Matcher

## 1. Pipeline
1. `mail:fetch` pulls Gmail messages (`gmail.users.messages.list/get`) and stores raw RFC822 `.eml` files.
2. `mail:process` loads a stored email, runs quote detection, extracts line items from text/html/xlsx/pdf, normalizes and matches against local catalog index.
3. `export:xlsx` renders per-email result table.

## 2. Connectors
- Active MVP connector: `connectors/gmail`.
- `connectors/imap` is scaffolded with `imapflow` and the same contract for future Yandex/IMAP mailboxes.

## 3. Catalog Sync
- Full sync: `GET /api/v1/product/scroll` with iterative `scrollId` until end.
- Incremental sync: same endpoint with exactly one filter per run: `hour_price` OR `hour_stock` OR `day`.
- Full tree refresh: `GET /api/v1/catalog/full-tree/` persisted once per ~30 days.
- API limiter: local limiter with default 5 req/sec (below 10 req/sec hard limit).

## 4. Matching Strategy
1. Exact by codes (`articul`, `syncUid`, `flatCodes.*`, `analogCodes`) => high confidence.
2. Exact by normalized `header`.
3. Fuzzy by token candidate generation + dice/token score.
4. REVIEW safety rules:
   - ambiguous top candidates,
   - low-confidence fuzzy,
   - qty missing/invalid (`qty <= 0`).

## 5. Confidence thresholds
- `OK` when `score >= MATCH_OK_THRESHOLD` and `(top1-top2) >= MATCH_GAP_THRESHOLD`.
- `REVIEW` when `MATCH_REVIEW_THRESHOLD <= score < MATCH_OK_THRESHOLD` or ambiguity.
- `NOT_FOUND` otherwise.

## 6. Storage model
SQLite tables:
- `products`
- `emails`
- `extractions`
- `matches`
- `runs`
- `metadata`

Idempotency:
- raw email content is hashed (`sha256`),
- processing clears and rewrites extraction/match rows for a given email,
- repeated processing of unchanged catalog/email yields same result.

## 7. Adding a new connector
1. Implement `MailConnector` in `connectors/<provider>`.
2. Return normalized `GmailFetchedMessage` payload (`provider`, `messageId`, headers, raw bytes).
3. Wire CLI `mail:fetch --provider=<provider>` selection.
4. Keep downstream pipeline unchanged (storage and processing are provider-agnostic).
