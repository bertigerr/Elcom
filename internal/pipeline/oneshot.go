package pipeline

import (
	"fmt"
	"os"

	"elcom/internal"
)

func ExtractItemsFromInput(inputType string, input string) ([]internal.ExtractionItem, error) {
	switch inputType {
	case "email_text":
		return parseEmailText(input), nil
	case "email_table":
		return parseEmailHTMLTable(input), nil
	case "xlsx":
		blob, err := os.ReadFile(input)
		if err != nil {
			return nil, err
		}
		return parseXLSX(blob)
	case "pdf":
		blob, err := os.ReadFile(input)
		if err != nil {
			return nil, err
		}
		return parsePDF(blob)
	default:
		return nil, fmt.Errorf("unsupported input type: %s", inputType)
	}
}
