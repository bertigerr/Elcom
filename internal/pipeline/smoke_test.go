package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"elcom/internal"
	"elcom/internal/config"
	"elcom/internal/storage"
)

func TestSmokeEmailToXLSX(t *testing.T) {
	tmp := t.TempDir()
	db, err := storage.Open(filepath.Join(tmp, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	products := []internal.ProductRecord{
		{ID: 100, Header: "Кабель ВВГнг 3x2.5", SyncUID: strp("sync-100"), Articul: strp("ELC100"), RawJSON: `{}`},
		{ID: 101, Header: "Провод ПВС 2x1.5", SyncUID: strp("sync-101"), Articul: strp("ELC101"), RawJSON: `{}`},
	}
	if err := db.UpsertProducts(products); err != nil {
		t.Fatal(err)
	}

	rawSrc := filepath.Join("testdata", "sample_quote.eml")
	rawBlob, err := os.ReadFile(rawSrc)
	if err != nil {
		t.Fatal(err)
	}
	rawPath := filepath.Join(tmp, "fixture.eml")
	if err := os.WriteFile(rawPath, rawBlob, 0o644); err != nil {
		t.Fatal(err)
	}

	email, err := db.UpsertEmail("gmail", "<fixture-1@example.com>", "Заявка", "customer@example.com", "2026-02-08T00:00:00Z", "hash", rawPath, "fetched")
	if err != nil {
		t.Fatal(err)
	}

	cfg, _ := config.Load()
	proc := NewProcessingService(db, cfg)
	res, err := proc.ProcessEmail(email)
	if err != nil {
		t.Fatal(err)
	}
	if res.Processed == 0 {
		t.Fatal("no lines processed")
	}

	rows, err := db.GetExportRows(email.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("no export rows")
	}

	out := filepath.Join(tmp, "result.xlsx")
	if err := ExportRowsToXLSX(rows, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
}

func strp(v string) *string { return &v }
