import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ElcomClient } from '../../src/catalog/elcomClient.js';
import { config } from '../../src/config.js';

describe('ElcomClient', () => {
  beforeEach(() => {
    config.elcomApiToken = 'test-token';
    config.elcomApiBaseUrl = 'https://example.test/api/v1';
    config.elcomRateLimitRps = 1000;
  });

  it('handles scroll pagination and retry', async () => {
    const responses = [
      { status: 500, body: { message: 'fail' } },
      {
        status: 200,
        body: {
          success: true,
          data: {
            scrollId: 'abc',
            products: [
              { id: 1, header: 'Кабель 1', articul: 'A1', flatCodes: {} },
            ],
          },
        },
      },
      {
        status: 200,
        body: {
          success: true,
          data: {
            scrollId: null,
            products: [
              { id: 2, header: 'Кабель 2', articul: 'A2', flatCodes: {} },
            ],
          },
        },
      },
    ];

    const fetchMock = vi.fn(async () => {
      const next = responses.shift();
      if (!next) throw new Error('No more responses');
      return {
        ok: next.status >= 200 && next.status < 300,
        status: next.status,
        json: async () => next.body,
        text: async () => JSON.stringify(next.body),
      } as Response;
    });

    vi.stubGlobal('fetch', fetchMock);

    const client = new ElcomClient();
    const products = await client.getProductsScrollAll();

    expect(products.length).toBe(2);
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });
});
