package listener

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"elcom/internal/config"
	"elcom/internal/connectors"
	gmailconnector "elcom/internal/connectors/gmail"
	imapconnector "elcom/internal/connectors/imap"
	"elcom/internal/pipeline"
	"elcom/internal/storage"
)

type Service struct {
	db  *storage.DB
	cfg config.Config
}

func NewService(db *storage.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) Run(ctx context.Context) error {
	for {
		if err := s.runCycle(ctx); err != nil {
			fmt.Printf("listener cycle error: %v\n", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(s.cfg.MailListenerIntervalSec) * time.Second):
		}
	}
}

func (s *Service) runCycle(ctx context.Context) error {
	provider := strings.ToLower(strings.TrimSpace(s.cfg.MailListenerProvider))
	mailConnector, err := s.makeConnector(provider)
	if err != nil {
		return err
	}

	fetchService := connectors.NewFetchService(s.db, s.cfg.RawMailDir, mailConnector)
	fetchResult, err := fetchService.FetchAndStore(s.cfg.MailListenerLabel, s.cfg.MailListenerFetchMax)
	if err != nil {
		return err
	}

	processor := pipeline.NewProcessingService(s.db, s.cfg)
	processedEmails, _, err := processor.ProcessPending(s.cfg.MailListenerProcessBatch, provider)
	if err != nil {
		return err
	}

	if s.cfg.MailListenerAutoExport {
		if err := s.exportProcessed(provider); err != nil {
			return err
		}
	}

	fmt.Printf("listener cycle done provider=%s fetched=%d stored=%d processed=%d\n", provider, fetchResult.Fetched, fetchResult.Stored, processedEmails)
	_ = ctx
	return nil
}

func (s *Service) exportProcessed(provider string) error {
	emails, err := s.db.ListEmailsByStatus("processed", 200)
	if err != nil {
		return err
	}

	for _, email := range emails {
		if email.Provider != provider {
			continue
		}
		rows, err := s.db.GetExportRows(email.ID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			continue
		}
		filename := fmt.Sprintf("%d_%s.xlsx", email.ID, sanitizeMessageID(email.MessageID))
		outputPath := filepath.Join(s.cfg.OutputDir, "listener", filename)
		if err := pipeline.ExportRowsToXLSX(rows, outputPath); err != nil {
			return err
		}
		_ = s.db.UpdateEmailStatus(email.ID, "exported")
	}
	return nil
}

func (s *Service) makeConnector(provider string) (connectors.MailConnector, error) {
	switch provider {
	case "gmail":
		return gmailconnector.NewConnector(s.cfg)
	case "imap":
		return imapconnector.NewConnector(s.cfg)
	default:
		return nil, fmt.Errorf("unsupported listener provider: %s", provider)
	}
}

func sanitizeMessageID(input string) string {
	repl := strings.NewReplacer("<", "_", ">", "_", ":", "_", "/", "_", "\\", "_", "|", "_", "?", "_", "*", "_", " ", "_")
	out := repl.Replace(input)
	if len(out) > 120 {
		out = out[:120]
	}
	return out
}
