package connectors

import (
	"elcom/internal/storage"
)

type FetchService struct {
	db        *storage.DB
	connector MailConnector
	store     *MailStoreService
}

type FetchResult struct {
	Fetched int
	Stored  int
}

func NewFetchService(db *storage.DB, rawMailDir string, connector MailConnector) *FetchService {
	return &FetchService{
		db:        db,
		connector: connector,
		store:     NewMailStoreService(db, rawMailDir),
	}
}

func (s *FetchService) FetchAndStore(label string, max int) (FetchResult, error) {
	messages, err := s.connector.FetchInbox(label, max)
	if err != nil {
		return FetchResult{}, err
	}

	stored := 0
	for _, msg := range messages {
		if _, err := s.store.Store(msg); err != nil {
			return FetchResult{}, err
		}
		stored++
	}

	return FetchResult{Fetched: len(messages), Stored: stored}, nil
}
