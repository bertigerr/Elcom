package pipeline

import "testing"

func TestParseEmailHTMLTable(t *testing.T) {
	html := `<table><tr><th>Наименование</th><th>Кол-во</th><th>Ед</th></tr><tr><td>ВВГнг 3х2.5</td><td>10</td><td>шт</td></tr></table>`
	items := parseEmailHTMLTable(html)
	if len(items) != 1 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].Qty == nil || *items[0].Qty != 10 {
		t.Fatalf("qty bad")
	}
}
