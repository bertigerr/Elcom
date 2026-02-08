package util

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	unitPattern   = regexp.MustCompile(`(?i)(шт|штук|pcs|pc|м\.?|метр|kg|кг|уп\.?|компл\.?)`)
	numberPattern = regexp.MustCompile(`(?i)(?:^|[^0-9.,])(\d{1,3}(?:[\s.,]\d{3})+|\d+(?:[.,]\d+)?)`)
)

type ParsedQty struct {
	Qty    *float64
	Unit   *string
	QtyRaw *string
}

func ParseQty(input string) ParsedQty {
	line := strings.ReplaceAll(input, "\u00A0", " ")

	qtyRaw := ""
	qtyToken := ""

	withUnit := regexp.MustCompile(`(?i)(?:^|[^0-9.,])(\d{1,3}(?:[\s.,]\d{3})+|\d+(?:[.,]\d+)?)\s*(шт|штук|pcs|pc|м\.?|метр|kg|кг|уп\.?|компл\.?)`)
	wm := withUnit.FindAllStringSubmatch(line, -1)
	if len(wm) > 0 {
		last := wm[len(wm)-1]
		qtyRaw = strings.TrimSpace(last[1] + " " + last[2])
		qtyToken = strings.TrimSpace(last[1])
	} else {
		nm := numberPattern.FindAllStringSubmatch(line, -1)
		if len(nm) > 0 {
			last := nm[len(nm)-1]
			qtyRaw = strings.TrimSpace(last[1])
			qtyToken = strings.TrimSpace(last[1])
		}
	}

	var qtyPtr *float64
	if qtyToken != "" {
		norm := normalizeNumericToken(qtyToken)
		if parsed, err := strconv.ParseFloat(norm, 64); err == nil {
			qtyPtr = FloatPtr(parsed)
		}
	}

	var unitPtr *string
	if um := unitPattern.FindStringSubmatch(line); len(um) > 1 {
		u := normalizeUnit(um[1])
		unitPtr = &u
	}

	var qtyRawPtr *string
	if qtyRaw != "" {
		qtyRawPtr = &qtyRaw
	}

	return ParsedQty{Qty: qtyPtr, Unit: unitPtr, QtyRaw: qtyRawPtr}
}

func normalizeUnit(unit string) string {
	u := strings.ToLower(strings.TrimSpace(unit))
	switch u {
	case "шт", "штук", "pcs", "pc":
		return "шт"
	case "м", "м.", "метр":
		return "м"
	case "kg", "кг":
		return "кг"
	case "уп", "уп.":
		return "уп"
	default:
		return u
	}
}

func normalizeNumericToken(token string) string {
	compact := strings.ReplaceAll(token, " ", "")
	if regexp.MustCompile(`^\d{1,3}(?:\.\d{3})+$`).MatchString(compact) {
		return strings.ReplaceAll(compact, ".", "")
	}
	if regexp.MustCompile(`^\d{1,3}(?:,\d{3})+$`).MatchString(compact) {
		return strings.ReplaceAll(compact, ",", "")
	}
	if strings.Contains(compact, ",") && !strings.Contains(compact, ".") {
		return strings.ReplaceAll(compact, ",", ".")
	}
	return compact
}
