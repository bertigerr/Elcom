# Elcom Mail -> Quote Parser & Catalog Matcher (MVP)

Production-oriented TypeScript MVP for:
- Gmail ingest (IMAP-ready architecture),
- quote extraction from email text/html/xlsx/pdf,
- catalog sync from Elcom API,
- local matching and XLSX export.

## API basis used
From `doc_elc_API.pdf`:
- `GET /api/v1/product/scroll` (+ `scrollId`) for full catalog sync.
- `GET /api/v1/product/scroll` with one of `hour_price|hour_stock|day` for incremental updates.
- `GET /api/v1/catalog/full-tree/` for rare tree refresh.

No web scraping is used.

## Quickstart
1. Install dependencies:
```bash
npm install
```

2. Configure env:
```bash
cp .env.example .env
```

3. Full catalog sync:
```bash
npm run catalog:initial-sync
```

4. Fetch Gmail messages:
```bash
npm run mail:fetch -- --provider=gmail --label=INBOX --max=50
```

5. Process one email:
```bash
npm run mail:process -- --provider=gmail --messageId='<message-id>'
```

Or process pending batch:
```bash
npm run mail:process -- --batch=20
```

6. Export result to XLSX:
```bash
npm run export:xlsx -- --emailId=1 --out=./out/result.xlsx
```

## Separate mail-listener microservice
Long-running listener (polling) that continuously:
1) fetches new emails,
2) processes quote lines,
3) writes XLSX outputs to `out/listener/`.

Start:
```bash
npm run mail:listen
```

Configure in `.env`:
- `MAIL_LISTENER_PROVIDER=gmail|imap`
- `MAIL_LISTENER_LABEL=INBOX`
- `MAIL_LISTENER_INTERVAL_SEC=30`
- `MAIL_LISTENER_FETCH_MAX=20`
- `MAIL_LISTENER_PROCESS_BATCH=20`
- `MAIL_LISTENER_AUTO_EXPORT=true`

For IMAP mode also set:
- `IMAP_HOST`, `IMAP_PORT`, `IMAP_SECURE`, `IMAP_USER`, `IMAP_PASSWORD`

## CLI commands
- `catalog:initial-sync`
- `catalog:incremental-sync --mode=hour_price|hour_stock|day`
- `mail:fetch --provider=gmail --label=INBOX --max=50`
- `mail:process --provider=gmail --messageId=...` (or `--batch=...`)
- `mail:listen` (separate microservice runtime)
- `export:xlsx --emailId=... --out=...`
- `run --input ... --type xlsx|pdf|email_text|email_table --output ...`

## Output columns
- `input_line_no`, `source`, `raw_line`
- `parsed_name_or_code`, `parsed_qty`, `parsed_unit`
- `match_status`, `confidence`, `match_reason`
- `product_id`, `product_syncUid`, `product_header`, `product_articul`, `unitHeader`
- `flat_elcom`, `flat_manufacturer`, `flat_raec`, `flat_pc`, `flat_etm`
- `candidate2_header`, `candidate2_score`

## Tests
```bash
npm test
```

Includes unit tests (qty, extraction, matcher) and integration smoke tests.
