import fs from 'node:fs';
import path from 'node:path';
import ExcelJS from 'exceljs';
import type { MatchExportRow } from '../../storage/db.js';

export async function exportRowsToXlsx(rows: MatchExportRow[], outputPath: string): Promise<void> {
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet('result');

  sheet.columns = [
    { header: 'input_line_no', key: 'input_line_no', width: 14 },
    { header: 'source', key: 'source', width: 16 },
    { header: 'raw_line', key: 'raw_line', width: 40 },
    { header: 'parsed_name_or_code', key: 'parsed_name_or_code', width: 36 },
    { header: 'parsed_qty', key: 'parsed_qty', width: 12 },
    { header: 'parsed_unit', key: 'parsed_unit', width: 12 },
    { header: 'match_status', key: 'match_status', width: 14 },
    { header: 'confidence', key: 'confidence', width: 12 },
    { header: 'match_reason', key: 'match_reason', width: 14 },
    { header: 'product_id', key: 'product_id', width: 12 },
    { header: 'product_syncUid', key: 'product_syncUid', width: 36 },
    { header: 'product_header', key: 'product_header', width: 40 },
    { header: 'product_articul', key: 'product_articul', width: 20 },
    { header: 'unitHeader', key: 'unitHeader', width: 12 },
    { header: 'flat_elcom', key: 'flat_elcom', width: 16 },
    { header: 'flat_manufacturer', key: 'flat_manufacturer', width: 20 },
    { header: 'flat_raec', key: 'flat_raec', width: 16 },
    { header: 'flat_pc', key: 'flat_pc', width: 16 },
    { header: 'flat_etm', key: 'flat_etm', width: 16 },
    { header: 'candidate2_header', key: 'candidate2_header', width: 40 },
    { header: 'candidate2_score', key: 'candidate2_score', width: 16 },
  ];

  for (const row of rows) {
    sheet.addRow(row);
  }

  sheet.getRow(1).font = { bold: true };
  sheet.views = [{ state: 'frozen', ySplit: 1 }];
  sheet.autoFilter = {
    from: 'A1',
    to: 'U1',
  };

  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  await workbook.xlsx.writeFile(outputPath);
}
