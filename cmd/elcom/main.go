package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"elcom/internal"
	"elcom/internal/catalog"
	"elcom/internal/config"
	"elcom/internal/connectors"
	gmailconnector "elcom/internal/connectors/gmail"
	imapconnector "elcom/internal/connectors/imap"
	"elcom/internal/listener"
	"elcom/internal/pipeline"
	"elcom/internal/storage"
)

func main() {
	cfg, err := config.Load()
	must(err)

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	db, err := storage.Open(cfg.DBPath)
	must(err)
	defer db.Close()

	cmd := os.Args[1]
	switch cmd {
	case "catalog:initial-sync":
		svc := catalog.NewSyncService(db, cfg)
		count, err := svc.InitialSync(context.Background())
		must(err)
		fmt.Printf("initial sync complete: %d products\n", count)
	case "catalog:incremental-sync":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		mode := fs.String("mode", "", "hour_price|hour_stock|day")
		_ = fs.Parse(os.Args[2:])
		if strings.TrimSpace(*mode) == "" {
			must(fmt.Errorf("--mode is required"))
		}
		svc := catalog.NewSyncService(db, cfg)
		count, err := svc.IncrementalSync(context.Background(), *mode)
		must(err)
		fmt.Printf("incremental sync complete mode=%s products=%d\n", *mode, count)
	case "mail:fetch":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		provider := fs.String("provider", "gmail", "gmail|imap")
		label := fs.String("label", "INBOX", "mailbox/label")
		max := fs.Int("max", 50, "max messages")
		_ = fs.Parse(os.Args[2:])
		conn, err := makeConnector(cfg, *provider)
		must(err)
		fetch := connectors.NewFetchService(db, cfg.RawMailDir, conn)
		result, err := fetch.FetchAndStore(*label, *max)
		must(err)
		fmt.Printf("mail fetch done provider=%s fetched=%d stored=%d\n", *provider, result.Fetched, result.Stored)
	case "mail:process":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		provider := fs.String("provider", "gmail", "gmail|imap")
		messageID := fs.String("messageId", "", "specific message-id")
		batch := fs.Int("batch", 20, "batch size")
		_ = fs.Parse(os.Args[2:])
		processor := pipeline.NewProcessingService(db, cfg)
		if strings.TrimSpace(*messageID) != "" {
			res, err := processor.ProcessByProviderMessageID(*provider, *messageID)
			must(err)
			fmt.Printf("processed email id=%d lines=%d\n", res.EmailID, res.Processed)
			return
		}
		processedEmails, processedLines, err := processor.ProcessPending(*batch, *provider)
		must(err)
		fmt.Printf("processed pending emails=%d lines=%d\n", processedEmails, processedLines)
	case "export:xlsx":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		emailID := fs.Int("emailId", 0, "internal email id")
		out := fs.String("out", "", "output xlsx path")
		_ = fs.Parse(os.Args[2:])
		if *emailID == 0 || strings.TrimSpace(*out) == "" {
			must(fmt.Errorf("--emailId and --out are required"))
		}
		rows, err := db.GetExportRows(*emailID)
		must(err)
		if len(rows) == 0 {
			must(fmt.Errorf("no export rows for emailId=%d", *emailID))
		}
		must(pipeline.ExportRowsToXLSX(rows, *out))
		fmt.Printf("exported %d rows to %s\n", len(rows), *out)
	case "mail:listen":
		s := listener.NewService(db, cfg)
		must(s.Run(context.Background()))
	case "run":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		input := fs.String("input", "", "input file path or raw text")
		inType := fs.String("type", "", "xlsx|pdf|email_text|email_table")
		output := fs.String("output", "", "output xlsx path")
		_ = fs.Parse(os.Args[2:])
		if *input == "" || *inType == "" || *output == "" {
			must(fmt.Errorf("--input --type --output are required"))
		}

		value := *input
		if (*inType == "xlsx" || *inType == "pdf") && !filepath.IsAbs(*input) {
			value = *input
		}
		items, err := pipeline.ExtractItemsFromInput(*inType, value)
		must(err)
		norm := pipeline.NormalizeItems(items)
		products, err := db.ListProducts()
		must(err)
		matcher := pipeline.NewMatcher(cfg, products)

		// Build temporary export rows for one-off run.
		exportRows := make([]internal.MatchExportRow, 0, len(norm))
		for _, item := range norm {
			match := matcher.Match(item)
			row := internal.MatchExportRow{
				InputLineNo:      item.LineNo,
				Source:           string(item.Source),
				RawLine:          item.RawLine,
				ParsedNameOrCode: item.NameOrCode,
				ParsedQty:        item.Qty,
				ParsedUnit:       item.Unit,
				MatchStatus:      string(match.Status),
				Confidence:       match.Confidence,
				MatchReason:      string(match.Reason),
			}
			if match.Product != nil {
				row.ProductID = match.Product.ID
				row.ProductSyncUID = match.Product.SyncUID
				row.ProductHeader = match.Product.Header
				row.ProductArticul = match.Product.Articul
				row.UnitHeader = match.Product.UnitHeader
				row.FlatElcom = match.Product.FlatCodes.Elcom
				row.FlatManufacturer = match.Product.FlatCodes.Manufacturer
				row.FlatRaec = match.Product.FlatCodes.Raec
				row.FlatPC = match.Product.FlatCodes.PC
				row.FlatEtm = match.Product.FlatCodes.Etm
			}
			if len(match.Candidates) > 1 {
				row.Candidate2Header = &match.Candidates[1].Header
				row.Candidate2Score = &match.Candidates[1].Score
			}
			exportRows = append(exportRows, row)
		}
		must(pipeline.ExportRowsToXLSX(exportRows, *output))
		fmt.Printf("run done rows=%d output=%s\n", len(exportRows), *output)
	default:
		usage()
		os.Exit(1)
	}
}

func makeConnector(cfg config.Config, provider string) (connectors.MailConnector, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "gmail":
		return gmailconnector.NewConnector(cfg)
	case "imap":
		return imapconnector.NewConnector(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func usage() {
	fmt.Println("usage: elcom <command>")
	fmt.Println("commands:")
	fmt.Println("  catalog:initial-sync")
	fmt.Println("  catalog:incremental-sync --mode=hour_price|hour_stock|day")
	fmt.Println("  mail:fetch --provider=gmail|imap --label=INBOX --max=50")
	fmt.Println("  mail:process --provider=gmail|imap [--messageId=...] [--batch=20]")
	fmt.Println("  mail:listen")
	fmt.Println("  export:xlsx --emailId=1 --out=./out/result.xlsx")
	fmt.Println("  run --input=... --type=xlsx|pdf|email_text|email_table --output=...xlsx")
}

func must(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
