import { config, requireEnv } from '../config.js';
import { logger } from '../logger.js';
import type { IncrementalMode, ProductRecord } from '../types.js';
import { toFlatCodes } from '../storage/db.js';
import { RateLimiter } from './rateLimiter.js';

interface ScrollPayload {
  products?: Array<Record<string, unknown>>;
  scrollId?: string | null;
  total?: number;
}

interface ApiResponse {
  success: boolean;
  message?: string;
  errors?: unknown;
  data?: ScrollPayload;
}

const RETRYABLE_STATUS = new Set([429, 500, 502, 503, 504]);

export class ElcomClient {
  private readonly limiter: RateLimiter;

  constructor() {
    this.limiter = new RateLimiter(config.elcomRateLimitRps);
  }

  private async fetchJson(path: string, query: Record<string, string | number | boolean | undefined>): Promise<ApiResponse> {
    await this.limiter.waitTurn();

    const token = requireEnv(config.elcomApiToken, 'ELCOM_API_TOKEN');
    const url = new URL(path, `${config.elcomApiBaseUrl.replace(/\/$/, '')}/`);
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== null && value !== '') {
        url.searchParams.set(key, String(value));
      }
    }

    let attempt = 0;
    // exponential backoff with jitter
    while (attempt < 5) {
      attempt += 1;
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          Authorization: `Bearer ${token}`,
          Accept: 'application/json',
        },
        signal: AbortSignal.timeout(config.elcomTimeoutMs),
      });

      if (!response.ok) {
        if (RETRYABLE_STATUS.has(response.status) && attempt < 5) {
          const backoff = 250 * 2 ** (attempt - 1) + Math.floor(Math.random() * 100);
          await new Promise((resolve) => setTimeout(resolve, backoff));
          continue;
        }
        throw new Error(`Elcom API error ${response.status}: ${await response.text()}`);
      }

      return (await response.json()) as ApiResponse;
    }

    throw new Error(`Elcom API retry limit reached for ${url.pathname}`);
  }

  async getProductsScrollAll(): Promise<ProductRecord[]> {
    const all: ProductRecord[] = [];
    let scrollId: string | undefined = undefined;
    const seenScrolls = new Set<string>();

    while (true) {
      const response = await this.fetchJson('product/scroll', { scrollId });
      if (!response.success) {
        throw new Error(`Elcom API unsuccessful response: ${JSON.stringify(response.errors)}`);
      }

      const payload = response.data ?? {};
      const batch = payload.products ?? [];
      for (const raw of batch) {
        all.push(toProductRecord(raw));
      }

      const nextScroll = payload.scrollId ?? undefined;
      if (!nextScroll || batch.length === 0 || seenScrolls.has(nextScroll)) {
        break;
      }

      seenScrolls.add(nextScroll);
      scrollId = nextScroll;
    }

    logger.info({ count: all.length }, 'Catalog full sync fetched');
    return all;
  }

  async getProductsIncremental(mode: IncrementalMode): Promise<ProductRecord[]> {
    const all: ProductRecord[] = [];
    let scrollId: string | undefined = undefined;
    const seenScrolls = new Set<string>();

    const filter: Record<string, string | number | boolean | undefined> = { scrollId };
    if (mode === 'day') {
      filter.day = config.incrementalLookbackDays;
    }
    if (mode === 'hour_price') {
      filter.hour_price = config.incrementalLookbackHours;
    }
    if (mode === 'hour_stock') {
      filter.hour_stock = config.incrementalLookbackHours;
    }

    while (true) {
      filter.scrollId = scrollId;
      const response = await this.fetchJson('product/scroll', filter);
      if (!response.success) {
        throw new Error(`Elcom API unsuccessful response: ${JSON.stringify(response.errors)}`);
      }

      const payload = response.data ?? {};
      const batch = payload.products ?? [];
      for (const raw of batch) {
        all.push(toProductRecord(raw));
      }

      const nextScroll = payload.scrollId ?? undefined;
      if (!nextScroll || batch.length === 0 || seenScrolls.has(nextScroll)) {
        break;
      }

      seenScrolls.add(nextScroll);
      scrollId = nextScroll;
    }

    logger.info({ mode, count: all.length }, 'Catalog incremental sync fetched');
    return all;
  }

  async getCatalogFullTree(): Promise<Record<string, unknown>> {
    const response = await this.fetchJson('catalog/full-tree/', {});
    if (!response.success) {
      throw new Error(`Elcom full-tree fetch failed: ${JSON.stringify(response.errors)}`);
    }
    return {
      message: response.message,
      data: response.data,
      fetchedAt: new Date().toISOString(),
    };
  }
}

function toStringOrNull(value: unknown): string | null {
  if (value == null) {
    return null;
  }
  return String(value);
}

function toNumberOrNull(value: unknown): number | null {
  if (value == null) {
    return null;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function toProductRecord(raw: Record<string, unknown>): ProductRecord {
  const analog = (raw.analogCodes as unknown[] | null | undefined) ?? [];
  const analogCodes = analog.map((v) => String(v)).filter(Boolean);
  const header = String(raw.header ?? '').trim();
  if (!header) {
    throw new Error(`Product payload missing header: ${JSON.stringify(raw)}`);
  }

  return {
    id: Number(raw.id),
    syncUid: toStringOrNull(raw.syncUid),
    header,
    articul: toStringOrNull(raw.articul),
    unitHeader: toStringOrNull(raw.unitHeader),
    manufacturerHeader: toStringOrNull(raw.manufacturerHeader),
    multiplicityOrder: toNumberOrNull(raw.multiplicityOrder),
    analogCodes,
    flatCodes: toFlatCodes(raw),
    updatedAt: toStringOrNull(raw.updatedAt),
    raw,
  };
}
