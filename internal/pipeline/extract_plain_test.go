package pipeline

import "testing"

func TestParseEmailText(t *testing.T) {
	text := "\nВВГнг 3х2.5 100 шт\nКабель NYM 10 м\n"
	items := parseEmailText(text)
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].Qty == nil || *items[0].Qty != 100 {
		t.Fatalf("qty1 bad")
	}
	if items[1].Qty == nil || *items[1].Qty != 10 {
		t.Fatalf("qty2 bad")
	}
}

func TestParseEmailTextCodeLike(t *testing.T) {
	items := parseEmailText("ELC0100203802 3 шт")
	if len(items) != 1 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].NameOrCode == nil || *items[0].NameOrCode != "ELC0100203802" {
		t.Fatalf("nameOrCode=%v", items[0].NameOrCode)
	}
}
