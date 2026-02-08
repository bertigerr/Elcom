package pipeline

import "strings"

type DetectResult struct {
	IsQuote bool
	Score   float64
	Reason  string
}

var detectKeywords = []string{"заявк", "кп", "коммерческ", "прошу", "нужно", "кол-во", "qty", "счет"}

func DetectQuoteRequest(subject, text, html string, attachmentNames []string) DetectResult {
	subject = strings.ToLower(subject)
	text = strings.ToLower(text)
	html = strings.ToLower(html)

	score := 0.0
	for _, kw := range detectKeywords {
		if strings.Contains(subject, kw) {
			score += 0.2
		}
		if strings.Contains(text, kw) || strings.Contains(html, kw) {
			score += 0.1
		}
	}

	qtyHits := countQtyPatterns(text)
	if qtyHits >= 2 {
		score += 0.4
	} else if qtyHits == 1 {
		score += 0.2
	}

	for _, name := range attachmentNames {
		ln := strings.ToLower(name)
		if strings.HasSuffix(ln, ".xlsx") || strings.HasSuffix(ln, ".xls") || strings.HasSuffix(ln, ".pdf") {
			score += 0.25
			break
		}
	}

	if strings.Contains(html, "<table") {
		score += 0.25
	}
	if score > 1 {
		score = 1
	}

	isQuote := score >= 0.45
	reason := "rules_negative"
	if isQuote {
		reason = "rules_positive"
	}

	return DetectResult{IsQuote: isQuote, Score: score, Reason: reason}
}

func countQtyPatterns(text string) int {
	count := 0
	for i := 0; i < len(text); i++ {
		if text[i] >= '0' && text[i] <= '9' {
			count++
			for i+1 < len(text) && text[i+1] >= '0' && text[i+1] <= '9' {
				i++
			}
		}
	}
	return count
}
