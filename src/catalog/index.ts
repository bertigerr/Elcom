import type { ProductRecord } from '../types.js';
import { normalizeCode, normalizeHeader, tokenize } from '../utils/text.js';

export interface ProductIndex {
  productsById: Map<number, ProductRecord>;
  byCode: Map<string, ProductRecord[]>;
  byHeader: Map<string, ProductRecord[]>;
  tokenToProductIds: Map<string, Set<number>>;
  normalizedHeaderById: Map<number, string>;
}

function pushMapValue<K, V>(map: Map<K, V[]>, key: K, value: V): void {
  const current = map.get(key);
  if (current) {
    current.push(value);
  } else {
    map.set(key, [value]);
  }
}

function addCode(byCode: Map<string, ProductRecord[]>, code: string | undefined | null, product: ProductRecord): void {
  if (!code) {
    return;
  }
  const normalized = normalizeCode(code);
  if (!normalized) {
    return;
  }
  pushMapValue(byCode, normalized, product);
}

export function buildProductIndex(products: ProductRecord[]): ProductIndex {
  const productsById = new Map<number, ProductRecord>();
  const byCode = new Map<string, ProductRecord[]>();
  const byHeader = new Map<string, ProductRecord[]>();
  const tokenToProductIds = new Map<string, Set<number>>();
  const normalizedHeaderById = new Map<number, string>();

  for (const product of products) {
    productsById.set(product.id, product);

    const normalizedHeader = normalizeHeader(product.header);
    normalizedHeaderById.set(product.id, normalizedHeader);
    pushMapValue(byHeader, normalizedHeader, product);

    addCode(byCode, product.articul, product);
    addCode(byCode, product.syncUid, product);
    addCode(byCode, product.flatCodes.elcom, product);
    addCode(byCode, product.flatCodes.manufacturer, product);
    addCode(byCode, product.flatCodes.raec, product);
    addCode(byCode, product.flatCodes.pc, product);
    addCode(byCode, product.flatCodes.etm, product);

    for (const analog of product.analogCodes ?? []) {
      addCode(byCode, analog, product);
    }

    for (const token of tokenize(product.header)) {
      if (!tokenToProductIds.has(token)) {
        tokenToProductIds.set(token, new Set<number>());
      }
      tokenToProductIds.get(token)?.add(product.id);
    }
  }

  return {
    productsById,
    byCode,
    byHeader,
    tokenToProductIds,
    normalizedHeaderById,
  };
}
