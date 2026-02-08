package pipeline

import (
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"

	"elcom/internal"
)

func ExportRowsToXLSX(rows []internal.MatchExportRow, outputPath string) error {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)

	headers := []string{
		"input_line_no", "source", "raw_line", "parsed_name_or_code", "parsed_qty", "parsed_unit",
		"match_status", "confidence", "match_reason",
		"product_id", "product_syncUid", "product_header", "product_articul", "unitHeader",
		"flat_elcom", "flat_manufacturer", "flat_raec", "flat_pc", "flat_etm",
		"candidate2_header", "candidate2_score",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	for i, row := range rows {
		r := i + 2
		set := func(col int, value any) {
			cell, _ := excelize.CoordinatesToCellName(col, r)
			_ = f.SetCellValue(sheet, cell, value)
		}

		set(1, row.InputLineNo)
		set(2, row.Source)
		set(3, row.RawLine)
		set(4, derefString(row.ParsedNameOrCode))
		set(5, derefFloat(row.ParsedQty))
		set(6, derefString(row.ParsedUnit))
		set(7, row.MatchStatus)
		set(8, row.Confidence)
		set(9, row.MatchReason)
		set(10, derefInt(row.ProductID))
		set(11, derefString(row.ProductSyncUID))
		set(12, derefString(row.ProductHeader))
		set(13, derefString(row.ProductArticul))
		set(14, derefString(row.UnitHeader))
		set(15, derefString(row.FlatElcom))
		set(16, derefString(row.FlatManufacturer))
		set(17, derefString(row.FlatRaec))
		set(18, derefString(row.FlatPC))
		set(19, derefString(row.FlatEtm))
		set(20, derefString(row.Candidate2Header))
		set(21, derefFloat(row.Candidate2Score))
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	return f.SaveAs(outputPath)
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefFloat(v *float64) any {
	if v == nil {
		return ""
	}
	return *v
}

func derefInt(v *int) any {
	if v == nil {
		return ""
	}
	return *v
}
