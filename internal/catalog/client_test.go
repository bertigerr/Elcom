package catalog

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"elcom/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGetProductsScrollAllWithRetry(t *testing.T) {
	attempt := 0

	cfg, _ := config.Load()
	cfg.ElcomAPIToken = "test"
	cfg.ElcomAPIBaseURL = "https://example.test/api/v1"
	cfg.ElcomRateLimitRPS = 1000

	client := NewClient(cfg)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/api/v1/product/scroll" {
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
			attempt++
			if attempt == 1 {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"boom"}`)),
					Header:     make(http.Header),
				}, nil
			}

			payload := map[string]any{"success": true, "data": map[string]any{"products": []map[string]any{}, "scrollId": nil}}
			if attempt == 2 {
				payload = map[string]any{"success": true, "data": map[string]any{"products": []map[string]any{{"id": 1, "header": "Кабель 1", "flatCodes": map[string]any{}}}, "scrollId": "abc"}}
			}
			if attempt == 3 {
				payload = map[string]any{"success": true, "data": map[string]any{"products": []map[string]any{{"id": 2, "header": "Кабель 2", "flatCodes": map[string]any{}}}, "scrollId": nil}}
			}
			blob, _ := json.Marshal(payload)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(blob))),
				Header:     make(http.Header),
			}, nil
		}),
	}

	products, err := client.GetProductsScrollAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 2 {
		t.Fatalf("len=%d", len(products))
	}
}
