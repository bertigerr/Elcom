package pipeline

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/jhillyerd/enmime"
	pdf "github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"

	"elcom/internal"
	"elcom/internal/util"
)

var ignorePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^--+$`),
	regexp.MustCompile(`(?i)^спасибо`),
	regexp.MustCompile(`(?i)^с уважением`),
	regexp.MustCompile(`(?i)^тел[:\s]`),
	regexp.MustCompile(`(?i)^e-?mail[:\s]`),
	regexp.MustCompile(`(?i)^http`),
}

func ExtractItemsFromEmailRaw(raw []byte) ([]internal.ExtractionItem, string, string, []string, error) {
	env, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		return nil, "", "", nil, err
	}

	items := make([]internal.ExtractionItem, 0)
	if env.Text != "" {
		items = append(items, parseEmailText(env.Text)...)
	}
	if env.HTML != "" {
		items = append(items, parseEmailHTMLTable(env.HTML)...)
	}

	attachmentNames := make([]string, 0, len(env.Attachments))
	for _, att := range env.Attachments {
		filename := strings.TrimSpace(att.FileName)
		if filename == "" {
			filename = "attachment"
		}
		attachmentNames = append(attachmentNames, filename)
		lower := strings.ToLower(filename)

		if strings.HasSuffix(lower, ".xlsx") || strings.HasSuffix(lower, ".xls") {
			extra, err := parseXLSX(att.Content)
			if err == nil {
				for i := range extra {
					if extra[i].Meta == nil {
						extra[i].Meta = map[string]any{}
					}
					extra[i].Meta["attachment"] = filename
				}
				items = append(items, extra...)
			}
		}
		if strings.HasSuffix(lower, ".pdf") {
			extra, err := parsePDF(att.Content)
			if err == nil {
				for i := range extra {
					if extra[i].Meta == nil {
						extra[i].Meta = map[string]any{}
					}
					extra[i].Meta["attachment"] = filename
				}
				items = append(items, extra...)
			}
		}
	}

	items = dedupeItems(items)
	for i := range items {
		items[i].LineNo = i + 1
	}

	return items, env.GetHeader("Subject"), env.Text, attachmentNames, nil
}

func parseEmailText(text string) []internal.ExtractionItem {
	lines := splitLines(text)
	out := make([]internal.ExtractionItem, 0, len(lines))
	lineNo := 0
	for _, line := range lines {
		lineNo++
		item := lineToExtractionItem(internal.SourceEmailText, lineNo, line)
		if item == nil {
			continue
		}
		hasLetters := regexp.MustCompile(`[A-Za-zА-Яа-я]`).MatchString(item.RawLine)
		hasQty := item.Qty != nil
		if !hasLetters || (!hasQty && len(item.RawLine) < 8) {
			continue
		}
		out = append(out, *item)
	}
	return out
}

func parseEmailHTMLTable(html string) []internal.ExtractionItem {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	out := []internal.ExtractionItem{}
	globalLine := 0
	doc.Find("table").Each(func(_ int, table *goquery.Selection) {
		rows := table.Find("tr")
		if rows.Length() < 2 {
			return
		}

		headers := []string{}
		rows.First().Find("th,td").Each(func(_ int, cell *goquery.Selection) {
			headers = append(headers, strings.ToLower(strings.TrimSpace(cell.Text())))
		})

		nameIdx := findHeaderIndex(headers, []string{"наименование", "товар", "позиция", "номенклатура", "name", "product"})
		qtyIdx := findHeaderIndex(headers, []string{"кол", "qty", "кол-во", "количество", "quantity"})
		unitIdx := findHeaderIndex(headers, []string{"ед", "unit", "изм"})

		rows.Slice(1, rows.Length()).Each(func(_ int, row *goquery.Selection) {
			cells := []string{}
			row.Find("th,td").Each(func(_ int, cell *goquery.Selection) {
				cells = append(cells, normalizeSpaces(cell.Text()))
			})
			if len(cells) == 0 {
				return
			}

			nameCell := pickCell(cells, nameIdx, 0)
			qtyCell := ""
			if qtyIdx >= 0 && qtyIdx < len(cells) {
				qtyCell = cells[qtyIdx]
			} else {
				for _, c := range cells {
					if regexp.MustCompile(`\d`).MatchString(c) {
						qtyCell = c
						break
					}
				}
			}
			unitCell := pickCell(cells, unitIdx, -1)

			parsed := util.ParseQty(qtyCell)
			rawLine := strings.Join(cells, " | ")
			if strings.TrimSpace(nameCell) == "" || (parsed.Qty == nil && !regexp.MustCompile(`\d`).MatchString(rawLine)) {
				return
			}

			globalLine++
			item := internal.ExtractionItem{
				LineNo:     globalLine,
				Source:     internal.SourceEmailHTMLTable,
				RawLine:    rawLine,
				NameOrCode: util.StringPtr(nameCell),
				Qty:        parsed.Qty,
				Unit:       parsed.Unit,
				Meta:       map[string]any{"row": cells},
			}
			if unitCell != "" {
				item.Unit = util.StringPtr(unitCell)
			}
			out = append(out, item)
		})
	})

	return out
}

func parseXLSX(content []byte) ([]internal.ExtractionItem, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lineNo := 0
	out := []internal.ExtractionItem{}
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		if len(rows) == 0 {
			continue
		}

		nameIdx, qtyIdx, unitIdx := -1, -1, -1
		for i, row := range rows {
			cells := normalizeCells(row)
			if len(cells) == 0 {
				continue
			}
			if i < 3 && nameIdx < 0 {
				nameIdx, qtyIdx, unitIdx = inferXLSColumns(cells)
				if nameIdx >= 0 || qtyIdx >= 0 {
					continue
				}
			}

			if nameIdx < 0 {
				nameIdx, qtyIdx, unitIdx = 0, 1, 2
			}
			name := pickCell(cells, nameIdx, 0)
			qtyCell := pickCell(cells, qtyIdx, -1)
			if qtyCell == "" {
				qtyCell = strings.Join(cells, " ")
			}
			parsed := util.ParseQty(qtyCell)
			if strings.TrimSpace(name) == "" || parsed.Qty == nil {
				continue
			}

			lineNo++
			item := internal.ExtractionItem{
				LineNo:     lineNo,
				Source:     internal.SourceXLSX,
				RawLine:    strings.Join(cells, " | "),
				NameOrCode: util.StringPtr(name),
				Qty:        parsed.Qty,
				Unit:       parsed.Unit,
				Meta:       map[string]any{"sheet": sheet, "rowNumber": i + 1},
			}
			if unit := pickCell(cells, unitIdx, -1); unit != "" {
				item.Unit = util.StringPtr(unit)
			}
			out = append(out, item)
		}
	}

	return out, nil
}

func parsePDF(content []byte) ([]internal.ExtractionItem, error) {
	r, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, err
	}

	out := []internal.ExtractionItem{}
	lineNo := 0
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		for _, line := range splitLines(text) {
			lineNo++
			item := lineToExtractionItem(internal.SourcePDF, lineNo, line)
			if item == nil {
				continue
			}
			if item.NameOrCode == nil || item.Qty == nil {
				continue
			}
			out = append(out, *item)
		}
	}
	return out, nil
}

func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	parts := strings.Split(text, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func lineToExtractionItem(source internal.ItemSource, lineNo int, rawLine string) *internal.ExtractionItem {
	compact := normalizeSpaces(rawLine)
	if compact == "" || isLikelyNoise(compact) {
		return nil
	}

	parsed := util.ParseQty(compact)
	noQty := compact
	if parsed.QtyRaw != nil {
		idx := strings.LastIndex(noQty, *parsed.QtyRaw)
		if idx >= 0 {
			noQty = noQty[:idx] + " " + noQty[idx+len(*parsed.QtyRaw):]
		}
	}

	name := regexp.MustCompile(`(?i)\b(шт|штук|pcs|pc|м\.?|метр|kg|кг|уп\.?|компл\.?)\b`).ReplaceAllString(noQty, " ")
	name = regexp.MustCompile(`[;|]+`).ReplaceAllString(name, " ")
	name = normalizeSpaces(name)

	nameOrCode := name
	if len([]rune(nameOrCode)) <= 1 {
		nameOrCode = compact
	}

	item := internal.ExtractionItem{
		LineNo:     lineNo,
		Source:     source,
		RawLine:    compact,
		NameOrCode: util.StringPtr(nameOrCode),
		Qty:        parsed.Qty,
		Unit:       parsed.Unit,
		Meta:       map[string]any{},
	}
	if parsed.QtyRaw != nil {
		item.Meta["qtyRaw"] = *parsed.QtyRaw
	}
	return &item
}

func normalizeSpaces(input string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(input, " "))
}

func isLikelyNoise(line string) bool {
	for _, re := range ignorePatterns {
		if re.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

func dedupeItems(items []internal.ExtractionItem) []internal.ExtractionItem {
	seen := map[string]struct{}{}
	out := make([]internal.ExtractionItem, 0, len(items))
	for _, item := range items {
		qtyKey := "null"
		if item.Qty != nil {
			qtyKey = fmt.Sprintf("%g", *item.Qty)
		}
		key := string(item.Source) + "|" + item.RawLine + "|" + qtyKey
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func findHeaderIndex(headers []string, probes []string) int {
	for i, h := range headers {
		for _, probe := range probes {
			if strings.Contains(h, probe) {
				return i
			}
		}
	}
	return -1
}

func pickCell(cells []string, idx int, fallback int) string {
	if idx >= 0 && idx < len(cells) {
		return strings.TrimSpace(cells[idx])
	}
	if fallback >= 0 && fallback < len(cells) {
		return strings.TrimSpace(cells[fallback])
	}
	return ""
}

func inferXLSColumns(headers []string) (nameIdx, qtyIdx, unitIdx int) {
	norm := make([]string, 0, len(headers))
	for _, h := range headers {
		norm = append(norm, strings.ToLower(h))
	}
	nameIdx = findHeaderIndex(norm, []string{"наимен", "товар", "номенк", "позиц", "name", "product"})
	qtyIdx = findHeaderIndex(norm, []string{"кол", "qty", "quantity"})
	unitIdx = findHeaderIndex(norm, []string{"ед", "unit", "изм"})
	return
}

func normalizeCells(row []string) []string {
	out := make([]string, 0, len(row))
	for _, c := range row {
		out = append(out, normalizeSpaces(c))
	}
	return out
}
