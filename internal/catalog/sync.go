package catalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"elcom/internal/config"
	"elcom/internal/storage"
)

type SyncService struct {
	db     *storage.DB
	client *Client
	cfg    config.Config
}

func NewSyncService(db *storage.DB, cfg config.Config) *SyncService {
	return &SyncService{db: db, client: NewClient(cfg), cfg: cfg}
}

func (s *SyncService) InitialSync(ctx context.Context) (int, error) {
	products, err := s.client.GetProductsScrollAll(ctx)
	if err != nil {
		return 0, err
	}
	if err := s.db.UpsertProducts(products); err != nil {
		return 0, err
	}
	_ = s.db.SetMetadata("catalog.last_initial_sync", time.Now().UTC().Format(time.RFC3339))
	if err := s.refreshFullTreeIfNeeded(ctx, true); err != nil {
		return 0, err
	}
	return len(products), nil
}

func (s *SyncService) IncrementalSync(ctx context.Context, mode string) (int, error) {
	products, err := s.client.GetProductsIncremental(ctx, mode)
	if err != nil {
		return 0, err
	}
	if len(products) > 0 {
		if err := s.db.UpsertProducts(products); err != nil {
			return 0, err
		}
	}
	_ = s.db.SetMetadata("catalog.last_incremental_sync."+mode, time.Now().UTC().Format(time.RFC3339))
	if err := s.refreshFullTreeIfNeeded(ctx, false); err != nil {
		return 0, err
	}
	return len(products), nil
}

func (s *SyncService) refreshFullTreeIfNeeded(ctx context.Context, force bool) error {
	const key = "catalog.last_full_tree_sync"
	last, err := s.db.GetMetadata(key)
	if err != nil {
		return err
	}

	if !force && last != nil {
		if parsed, err := time.Parse(time.RFC3339, *last); err == nil {
			if time.Since(parsed) < 30*24*time.Hour {
				return nil
			}
		}
	}

	tree, err := s.client.GetCatalogFullTree(ctx)
	if err != nil {
		return err
	}
	blob, _ := json.MarshalIndent(tree, "", "  ")
	treePath := filepath.Join(s.cfg.OutputDir, "catalog-full-tree.json")
	if err := os.MkdirAll(filepath.Dir(treePath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(treePath, blob, 0o644); err != nil {
		return err
	}
	return s.db.SetMetadata(key, time.Now().UTC().Format(time.RFC3339))
}
