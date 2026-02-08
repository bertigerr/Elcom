import fs from 'node:fs';
import path from 'node:path';
import { config } from '../config.js';
import { logger } from '../logger.js';
import type { IncrementalMode } from '../types.js';
import { AppDb } from '../storage/db.js';
import { ElcomClient } from './elcomClient.js';

export class CatalogSyncService {
  constructor(
    private readonly db: AppDb,
    private readonly client: ElcomClient,
  ) {}

  async initialSync(): Promise<{ products: number }> {
    const products = await this.client.getProductsScrollAll();
    this.db.upsertProducts(products);
    this.db.setMetadata('catalog.last_initial_sync', new Date().toISOString());
    await this.refreshFullTreeIfNeeded(true);

    logger.info({ products: products.length }, 'Initial sync completed');
    return { products: products.length };
  }

  async incrementalSync(mode: IncrementalMode): Promise<{ products: number }> {
    const products = await this.client.getProductsIncremental(mode);
    if (products.length > 0) {
      this.db.upsertProducts(products);
    }

    this.db.setMetadata(`catalog.last_incremental_sync.${mode}`, new Date().toISOString());
    await this.refreshFullTreeIfNeeded(false);

    logger.info({ products: products.length, mode }, 'Incremental sync completed');
    return { products: products.length };
  }

  private async refreshFullTreeIfNeeded(force: boolean): Promise<void> {
    const key = 'catalog.last_full_tree_sync';
    const last = this.db.getMetadata(key);
    const monthMs = 1000 * 60 * 60 * 24 * 30;

    if (!force && last) {
      const delta = Date.now() - new Date(last).getTime();
      if (delta < monthMs) {
        return;
      }
    }

    const tree = await this.client.getCatalogFullTree();
    const treePath = path.join(config.outputDir, 'catalog-full-tree.json');
    fs.mkdirSync(path.dirname(treePath), { recursive: true });
    fs.writeFileSync(treePath, JSON.stringify(tree, null, 2), 'utf8');
    this.db.setMetadata(key, new Date().toISOString());

    logger.info({ treePath }, 'Catalog full-tree refreshed');
  }
}
