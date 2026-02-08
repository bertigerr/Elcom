# DESIGN: Mail -> Quote Parser & Catalog Matcher (Go)

## 1. Pipeline
1. `mail:fetch` pulls messages from Gmail API or IMAP and stores raw `.eml` files.
2. `mail:process` loads stored email, runs quote detection, extracts line items from text/html/xlsx/pdf, normalizes and matches against local catalog index.
3. `export:xlsx` renders per-email result table.
4. `cmd/mail-listener` runs polling loop: fetch + process + auto-export continuously.

## 2. Connectors
- Gmail connector: OAuth refresh token + Gmail API (`users.messages.list/get`).
- IMAP connector: TLS IMAP with optional `\Seen` mark.
- Shared mail store service writes raw RFC822 and upserts email metadata idempotently.

## 3. Catalog Sync
- Full sync: `GET /api/v1/product/scroll` with iterative `scrollId`.
- Incremental sync: same endpoint with exactly one filter per run: `hour_price` OR `hour_stock` OR `day`.
- Full tree refresh: `GET /api/v1/catalog/full-tree/` persisted ~once per 30 days.
- API limiter: default 5 req/sec (< 10 req/sec hard limit).

## 4. Matching Strategy
1. Exact by codes (`articul`, `syncUid`, `flatCodes.*`, `analogCodes`) -> high confidence.
2. Exact by normalized `header`.
3. Fuzzy by token candidate generation + dice/token score.
4. REVIEW safety rules:
   - ambiguous candidates,
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
- raw email content hash (`sha256`) controls raw file naming,
- processing clears and rewrites extraction/match rows per email,
- repeated processing of unchanged catalog/email yields stable output.

## 7. Adding a new connector
1. Implement `connectors.MailConnector`.
2. Return normalized `internal.FetchedMailMessage`.
3. Wire connector selection in `cmd/elcom` and listener provider switch.
4. Keep downstream pipeline unchanged.
