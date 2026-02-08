import { describe, expect, it } from 'vitest';
import { parseEmailHtmlTable } from '../../src/pipeline/extract/htmlTable.js';

describe('parseEmailHtmlTable', () => {
  it('extracts by semantic headers', () => {
    const html = `
      <table>
        <tr><th>Наименование</th><th>Кол-во</th><th>Ед</th></tr>
        <tr><td>ВВГнг 3х2.5</td><td>10</td><td>шт</td></tr>
      </table>
    `;

    const items = parseEmailHtmlTable(html);
    expect(items.length).toBe(1);
    expect(items[0].qty).toBe(10);
    expect(items[0].unit).toBe('шт');
  });

  it('falls back when headers absent', () => {
    const html = `
      <table>
        <tr><td>abc</td><td>def</td></tr>
        <tr><td>Кабель NYM</td><td>7</td></tr>
      </table>
    `;

    const items = parseEmailHtmlTable(html);
    expect(items.length).toBe(1);
    expect(items[0].qty).toBe(7);
  });

  it('supports multiple rows', () => {
    const html = `
      <table>
        <tr><th>Товар</th><th>Quantity</th></tr>
        <tr><td>Провод ПВС</td><td>3</td></tr>
        <tr><td>Кабель КГ</td><td>4</td></tr>
      </table>
    `;

    const items = parseEmailHtmlTable(html);
    expect(items.length).toBe(2);
  });
});
