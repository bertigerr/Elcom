import ExcelJS from 'exceljs';
import type { ExtractionItem } from '../../types.js';
import { parseQty } from '../../utils/qty.js';

function inferColumns(headerCells: string[]): { nameIdx: number; qtyIdx: number; unitIdx: number } {
  const normalized = headerCells.map((v) => v.toLowerCase());
  const find = (probes: string[]) => normalized.findIndex((h) => probes.some((p) => h.includes(p)));

  const nameIdx = find(['наимен', 'товар', 'номенк', 'позиц', 'name', 'product']);
  const qtyIdx = find(['кол', 'qty', 'quantity']);
  const unitIdx = find(['ед', 'unit', 'изм']);

  return { nameIdx, qtyIdx, unitIdx };
}

export async function parseXlsx(buffer: Buffer | Uint8Array): Promise<ExtractionItem[]> {
  const workbook = new ExcelJS.Workbook();
  await (workbook.xlsx as unknown as { load: (input: unknown) => Promise<void> }).load(buffer);
  const out: ExtractionItem[] = [];
  let lineNo = 0;

  for (const sheet of workbook.worksheets) {
    let inferred: { nameIdx: number; qtyIdx: number; unitIdx: number } | null = null;

    sheet.eachRow((row, rowNumber) => {
      const cells: string[] = [];
      row.eachCell({ includeEmpty: true }, (cell, colNumber) => {
        const value = cell.value == null ? '' : String(cell.value);
        cells[colNumber - 1] = value.replace(/\s+/g, ' ').trim();
      });

      if (!cells.some(Boolean)) {
        return;
      }

      if (!inferred && rowNumber <= 3) {
        inferred = inferColumns(cells);
        return;
      }

      if (!inferred) {
        inferred = { nameIdx: 0, qtyIdx: 1, unitIdx: 2 };
      }

      const name = inferred.nameIdx >= 0 ? cells[inferred.nameIdx] : cells[0];
      const qtyCell = inferred.qtyIdx >= 0 ? cells[inferred.qtyIdx] : cells.find((c) => /\d/.test(c)) ?? '';
      const unitCell = inferred.unitIdx >= 0 ? cells[inferred.unitIdx] : null;

      if (!name && !qtyCell) {
        return;
      }

      const parsed = parseQty(qtyCell || cells.join(' '));
      if (!name || parsed.qty === null) {
        return;
      }

      lineNo += 1;
      out.push({
        lineNo,
        source: 'xlsx',
        rawLine: cells.join(' | '),
        nameOrCode: name,
        qty: parsed.qty,
        unit: unitCell || parsed.unit,
        meta: { sheet: sheet.name, rowNumber },
      });
    });
  }

  return out;
}

export async function parseXlsxFile(filePath: string): Promise<ExtractionItem[]> {
  const fs = await import('node:fs/promises');
  const buffer = await fs.readFile(filePath);
  return parseXlsx(buffer);
}
