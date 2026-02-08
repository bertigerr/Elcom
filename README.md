# Elcom Mail -> Quote Parser & Catalog Matcher (Go)

Production-oriented Go service for:
- mail ingest (Gmail API or IMAP),
- quote extraction from email text/html/xlsx/pdf text layer,
- catalog sync from Elcom API,
- local matching and XLSX export,
- standalone mail-listener microservice.

## API basis used
From `doc_elc_API.pdf`:
- `GET /api/v1/product/scroll` (+ `scrollId`) for full catalog sync.
- `GET /api/v1/product/scroll` with one of `hour_price|hour_stock|day` for incremental updates.
- `GET /api/v1/catalog/full-tree/` for rare tree refresh.

No web scraping is used.

## Build
```bash
go mod tidy
go build ./...
```

## Tests
```bash
go test ./...
```

## CLI commands
```bash
go run ./cmd/elcom -- catalog:initial-sync
go run ./cmd/elcom -- catalog:incremental-sync --mode=hour_price
go run ./cmd/elcom -- mail:fetch --provider=gmail --label=INBOX --max=50
go run ./cmd/elcom -- mail:process --provider=gmail --batch=20
go run ./cmd/elcom -- export:xlsx --emailId=1 --out=./out/result.xlsx
```

One-off run from input:
```bash
go run ./cmd/elcom -- run --input="Кабель ВВГнг 3x2.5 10 шт" --type=email_text --output=./out/quick.xlsx
```

## Standalone listener microservice
Runs continuous polling cycle: fetch -> process -> export.

```bash
go run ./cmd/mail-listener
```

## Environment
Copy and fill:
```bash
cp .env.example .env
```

Minimum for Gmail mode:
- `GMAIL_CLIENT_ID`
- `GMAIL_CLIENT_SECRET`
- `GMAIL_REFRESH_TOKEN`
- `MAIL_LISTENER_PROVIDER=gmail`

Minimum for IMAP mode:
- `IMAP_HOST`
- `IMAP_PORT`
- `IMAP_SECURE`
- `IMAP_USER`
- `IMAP_PASSWORD`
- `MAIL_LISTENER_PROVIDER=imap`

## Output columns
- `input_line_no`, `source`, `raw_line`
- `parsed_name_or_code`, `parsed_qty`, `parsed_unit`
- `match_status`, `confidence`, `match_reason`
- `product_id`, `product_syncUid`, `product_header`, `product_articul`, `unitHeader`
- `flat_elcom`, `flat_manufacturer`, `flat_raec`, `flat_pc`, `flat_etm`
- `candidate2_header`, `candidate2_score`
