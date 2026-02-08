package util

import "testing"

func TestParseQty(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  float64
	}{
		{name: "thousand with space", input: "Кабель 1 000 шт", want: 1000},
		{name: "decimal comma", input: "Провод 1,5 м", want: 1.5},
		{name: "decimal dot", input: "Провод 1.5 м", want: 1.5},
		{name: "thousand dot", input: "Кабель 1.000 шт", want: 1000},
		{name: "dimension and qty", input: "ВВГнг 3х2.5 100 шт", want: 100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed := ParseQty(tc.input)
			if parsed.Qty == nil {
				t.Fatalf("qty is nil")
			}
			if *parsed.Qty != tc.want {
				t.Fatalf("got %v want %v", *parsed.Qty, tc.want)
			}
		})
	}
}
