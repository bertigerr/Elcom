import ExcelJS from 'exceljs';
import { describe, expect, it } from 'vitest';
import { parseXlsx } from '../../src/pipeline/extract/xlsx.js';

async function buildWorkbook(rows: Array<Array<string | number>>): Promise<Buffer> {
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet('Sheet1');
  for (const row of rows) {
    sheet.addRow(row);
  }
  return Buffer.from(await workbook.xlsx.writeBuffer());
}

describe('parseXlsx', () => {
  it('extracts rows with one header row', async () => {
    const buffer = await buildWorkbook([
      ['Наименование', 'Кол-во', 'Ед.'],
      ['Кабель ВВГ', 10, 'шт'],
      ['Провод ПВС', 2, 'м'],
    ]);

    const items = await parseXlsx(buffer);
    expect(items.length).toBe(2);
  });

  it('works with two header rows', async () => {
    const buffer = await buildWorkbook([
      ['Заявка', '', ''],
      ['Наименование', 'Кол-во', 'Ед.'],
      ['Кабель КГ', 5, 'шт'],
    ]);

    const items = await parseXlsx(buffer);
    expect(items.length).toBe(1);
    expect(items[0].qty).toBe(5);
  });

  it('skips invalid rows', async () => {
    const buffer = await buildWorkbook([
      ['Наименование', 'Кол-во'],
      ['Текст без qty', ''],
      ['Кабель', 8],
    ]);

    const items = await parseXlsx(buffer);
    expect(items.length).toBe(1);
  });
});
