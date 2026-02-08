package util

import (
	"regexp"
	"strings"
)

var (
	reQuotes     = regexp.MustCompile(`["'` + "`" + `«»]`)
	reNonAllowed = regexp.MustCompile(`[^A-ZА-Я0-9X\-/\s.]`)
	reSpaces     = regexp.MustCompile(`\s+`)
)

func NormalizeHeader(input string) string {
	s := strings.ToUpper(input)
	s = strings.ReplaceAll(s, "Ё", "Е")
	repl := strings.NewReplacer("×", "X", "Х", "X", "х", "X", "*", "X", "ММ²", "MM2", "КВ.ММ", "MM2", "КВ ММ", "MM2", "MM²", "MM2")
	s = repl.Replace(s)
	s = reQuotes.ReplaceAllString(s, " ")
	s = reNonAllowed.ReplaceAllString(s, " ")
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func NormalizeCode(input string) string {
	s := strings.ToUpper(input)
	repl := strings.NewReplacer("×", "X", "Х", "X", "х", "X", "*", "X")
	s = repl.Replace(s)
	s = strings.ReplaceAll(s, " ", "")
	out := strings.Builder{}
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'А' && r <= 'Я') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/' || r == '.' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func Tokenize(input string) []string {
	norm := NormalizeHeader(input)
	parts := strings.Split(norm, " ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len([]rune(p)) >= 2 {
			out = append(out, p)
		}
	}
	return out
}

func LooksLikeCode(input string) bool {
	if len(strings.TrimSpace(input)) < 3 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, r := range input {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= 'А' && r <= 'Я') || (r >= 'а' && r <= 'я') {
			hasLetter = true
		}
		if r >= '0' && r <= '9' {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

func DiceCoefficient(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}

	pairs := func(s string) []string {
		r := []rune(s)
		if len(r) < 2 {
			return nil
		}
		out := make([]string, 0, len(r)-1)
		for i := 0; i < len(r)-1; i++ {
			out = append(out, string(r[i:i+2]))
		}
		return out
	}

	aPairs := pairs(a)
	bPairs := pairs(b)
	if len(aPairs) == 0 || len(bPairs) == 0 {
		return 0
	}

	bCount := map[string]int{}
	for _, p := range bPairs {
		bCount[p]++
	}
	inter := 0
	for _, p := range aPairs {
		if bCount[p] > 0 {
			inter++
			bCount[p]--
		}
	}

	return float64(2*inter) / float64(len(aPairs)+len(bPairs))
}
