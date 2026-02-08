import { load } from 'cheerio';
import type { ExtractionItem } from '../../types.js';
import { parseQty } from '../../utils/qty.js';

const NAME_HEADERS = ['наименование', 'товар', 'позиция', 'номенклатура', 'name', 'product'];
const QTY_HEADERS = ['кол', 'qty', 'кол-во', 'количество', 'quantity'];
const UNIT_HEADERS = ['ед', 'unit', 'изм'];

function findHeaderIndex(headers: string[], probes: string[]): number {
  return headers.findIndex((h) => probes.some((probe) => h.includes(probe)));
}

export function parseEmailHtmlTable(html: string): ExtractionItem[] {
  const $ = load(html);
  const out: ExtractionItem[] = [];
  let globalLine = 0;

  $('table').each((_, table) => {
    const rows = $(table).find('tr');
    if (rows.length < 2) {
      return;
    }

    const firstRowCells = $(rows[0])
      .find('th,td')
      .toArray()
      .map((c) => $(c).text().trim().toLowerCase());

    const nameIdx = findHeaderIndex(firstRowCells, NAME_HEADERS);
    const qtyIdx = findHeaderIndex(firstRowCells, QTY_HEADERS);
    const unitIdx = findHeaderIndex(firstRowCells, UNIT_HEADERS);

    rows.slice(1).each((__, row) => {
      const cells = $(row)
        .find('td,th')
        .toArray()
        .map((c) => $(c).text().replace(/\s+/g, ' ').trim());

      if (!cells.length) {
        return;
      }

      const nameCell = nameIdx >= 0 && nameIdx < cells.length ? cells[nameIdx] : cells[0];
      const qtyCell = qtyIdx >= 0 && qtyIdx < cells.length ? cells[qtyIdx] : cells.find((c) => /\d/.test(c)) ?? '';
      const unitCell = unitIdx >= 0 && unitIdx < cells.length ? cells[unitIdx] : null;

      const parsed = parseQty(qtyCell);
      const rawLine = cells.join(' | ');
      if (!nameCell || (!parsed.qty && !/\d/.test(rawLine))) {
        return;
      }

      globalLine += 1;
      out.push({
        lineNo: globalLine,
        source: 'email_html_table',
        rawLine,
        nameOrCode: nameCell,
        qty: parsed.qty,
        unit: unitCell || parsed.unit,
        meta: {
          row: cells,
        },
      });
    });
  });

  return out;
}
