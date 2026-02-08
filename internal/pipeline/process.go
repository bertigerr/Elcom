package pipeline

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"elcom/internal"
	"elcom/internal/config"
	"elcom/internal/storage"
)

type ProcessingService struct {
	db  *storage.DB
	cfg config.Config
}

func NewProcessingService(db *storage.DB, cfg config.Config) *ProcessingService {
	return &ProcessingService{db: db, cfg: cfg}
}

type ProcessResult struct {
	EmailID   int
	Processed int
}

func (s *ProcessingService) ProcessByProviderMessageID(provider, messageID string) (ProcessResult, error) {
	email, err := s.db.MustEmailByProviderMessageID(provider, messageID)
	if err != nil {
		return ProcessResult{}, err
	}
	return s.ProcessEmail(email)
}

func (s *ProcessingService) ProcessPending(limit int, provider string) (int, int, error) {
	pending, err := s.db.ListEmailsByStatus("fetched", limit)
	if err != nil {
		return 0, 0, err
	}
	processedEmails := 0
	processedLines := 0
	for _, email := range pending {
		if provider != "" && email.Provider != provider {
			continue
		}
		res, err := s.ProcessEmail(email)
		if err != nil {
			return processedEmails, processedLines, err
		}
		processedEmails++
		processedLines += res.Processed
	}
	return processedEmails, processedLines, nil
}

func (s *ProcessingService) ProcessEmail(email internal.EmailRow) (ProcessResult, error) {
	start := time.Now()
	raw, err := os.ReadFile(email.RawRef)
	if err != nil {
		return ProcessResult{}, err
	}

	items, subject, text, attachmentNames, err := ExtractItemsFromEmailRaw(raw)
	if err != nil {
		return ProcessResult{}, err
	}

	detect := DetectQuoteRequest(firstNonEmpty(subject, email.Subject), text, "", attachmentNames)
	if err := s.db.ClearEmailProcessing(email.ID); err != nil {
		return ProcessResult{}, err
	}

	if !detect.IsQuote {
		_ = s.db.UpdateEmailStatus(email.ID, "skipped")
		_ = s.db.InsertRun(traceID(), email.ID, map[string]float64{"totalMs": float64(time.Since(start).Milliseconds())}, map[string]int{"extracted": 0, "ok": 0, "review": 0, "notFound": 0})
		return ProcessResult{EmailID: email.ID, Processed: 0}, nil
	}

	normalized := NormalizeItems(items)
	products, err := s.db.ListProducts()
	if err != nil {
		return ProcessResult{}, err
	}
	matcher := NewMatcher(s.cfg, products)

	okCount, reviewCount, notFoundCount := 0, 0, 0
	for _, item := range normalized {
		match := matcher.Match(item)
		extractionID, err := s.db.InsertExtraction(email.ID, item.ExtractionItem)
		if err != nil {
			return ProcessResult{}, err
		}
		if err := s.db.InsertMatch(extractionID, match); err != nil {
			return ProcessResult{}, err
		}

		switch match.Status {
		case internal.MatchOK:
			okCount++
		case internal.MatchReview:
			reviewCount++
		case internal.MatchNotFound:
			notFoundCount++
		}
	}

	if err := s.db.UpdateEmailStatus(email.ID, "processed"); err != nil {
		return ProcessResult{}, err
	}
	_ = s.db.InsertRun(traceID(), email.ID, map[string]float64{"totalMs": float64(time.Since(start).Milliseconds())}, map[string]int{"extracted": len(normalized), "ok": okCount, "review": reviewCount, "notFound": notFoundCount})

	return ProcessResult{EmailID: email.ID, Processed: len(normalized)}, nil
}

func traceID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
