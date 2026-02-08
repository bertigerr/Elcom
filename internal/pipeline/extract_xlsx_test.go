package pipeline

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func mkXLSX(rows [][]any) []byte {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	buf := bytes.NewBuffer(nil)
	_, _ = f.WriteTo(buf)
	return buf.Bytes()
}

func TestParseXLSX(t *testing.T) {
	blob := mkXLSX([][]any{
		{"Наименование", "Кол-во", "Ед"},
		{"Кабель ВВГ", 10, "шт"},
		{"Провод ПВС", 2, "м"},
	})
	items, err := parseXLSX(blob)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
}
