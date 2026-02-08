package connectors

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"elcom/internal"
	"elcom/internal/storage"
)

type MailStoreService struct {
	db         *storage.DB
	rawMailDir string
}

func NewMailStoreService(db *storage.DB, rawMailDir string) *MailStoreService {
	return &MailStoreService{db: db, rawMailDir: rawMailDir}
}

func (s *MailStoreService) Store(msg internal.FetchedMailMessage) (internal.EmailRow, error) {
	hashBytes := sha256.Sum256(msg.Raw)
	hash := hex.EncodeToString(hashBytes[:])

	if err := os.MkdirAll(s.rawMailDir, 0o755); err != nil {
		return internal.EmailRow{}, err
	}

	rawPath := filepath.Join(s.rawMailDir, hash+".eml")
	if _, err := os.Stat(rawPath); os.IsNotExist(err) {
		if err := os.WriteFile(rawPath, msg.Raw, 0o644); err != nil {
			return internal.EmailRow{}, err
		}
	}

	return s.db.UpsertEmail(msg.Provider, msg.MessageID, msg.Subject, msg.From, msg.ReceivedAt, hash, rawPath, "fetched")
}
