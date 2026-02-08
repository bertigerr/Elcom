package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"elcom/internal"
	"elcom/internal/config"
	"elcom/internal/util"
)

type Client struct {
	cfg        config.Config
	httpClient *http.Client
	limiter    *RateLimiter
}

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Errors  json.RawMessage `json:"errors"`
	Data    json.RawMessage `json:"data"`
}

type scrollPayload struct {
	Products []map[string]any `json:"products"`
	ScrollID *string          `json:"scrollId"`
	Total    *int             `json:"total"`
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: time.Duration(cfg.ElcomTimeoutMs) * time.Millisecond},
		limiter:    NewRateLimiter(cfg.ElcomRateLimitRPS),
	}
}

func (c *Client) GetProductsScrollAll(ctx context.Context) ([]internal.ProductRecord, error) {
	return c.getProductsScroll(ctx, map[string]string{})
}

func (c *Client) GetProductsIncremental(ctx context.Context, mode string) ([]internal.ProductRecord, error) {
	params := map[string]string{}
	switch mode {
	case "day":
		params["day"] = strconv.Itoa(c.cfg.IncrementalLookbackDay)
	case "hour_price":
		params["hour_price"] = strconv.Itoa(c.cfg.IncrementalLookbackHrs)
	case "hour_stock":
		params["hour_stock"] = strconv.Itoa(c.cfg.IncrementalLookbackHrs)
	default:
		return nil, fmt.Errorf("unsupported incremental mode: %s", mode)
	}
	return c.getProductsScroll(ctx, params)
}

func (c *Client) GetCatalogFullTree(ctx context.Context) (map[string]any, error) {
	body, err := c.fetchJSON(ctx, "catalog/full-tree/", map[string]string{})
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) getProductsScroll(ctx context.Context, params map[string]string) ([]internal.ProductRecord, error) {
	all := make([]internal.ProductRecord, 0)
	seen := map[string]struct{}{}
	var scrollID string

	for {
		query := map[string]string{}
		for k, v := range params {
			query[k] = v
		}
		if scrollID != "" {
			query["scrollId"] = scrollID
		}

		body, err := c.fetchJSON(ctx, "product/scroll", query)
		if err != nil {
			return nil, err
		}

		var payload scrollPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}

		for _, raw := range payload.Products {
			product, err := toProductRecord(raw)
			if err != nil {
				continue
			}
			all = append(all, product)
		}

		if payload.ScrollID == nil || *payload.ScrollID == "" || len(payload.Products) == 0 {
			break
		}
		if _, ok := seen[*payload.ScrollID]; ok {
			break
		}
		seen[*payload.ScrollID] = struct{}{}
		scrollID = *payload.ScrollID
	}

	return all, nil
}

func (c *Client) fetchJSON(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	if strings.TrimSpace(c.cfg.ElcomAPIToken) == "" {
		return nil, errors.New("missing ELCOM_API_TOKEN")
	}

	baseURL := strings.TrimRight(c.cfg.ElcomAPIBaseURL, "/") + "/"
	u, err := url.Parse(baseURL + endpoint)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	for k, v := range params {
		if strings.TrimSpace(v) != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		c.limiter.WaitTurn()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.cfg.ElcomAPIToken)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if isRetryableStatus(resp.StatusCode) && attempt < 5 {
				backoff := time.Duration(250*(1<<(attempt-1))+rand.Intn(100)) * time.Millisecond
				time.Sleep(backoff)
				lastErr = fmt.Errorf("elcom status %d", resp.StatusCode)
				continue
			}
			return nil, fmt.Errorf("elcom api error: status=%d body=%s", resp.StatusCode, string(body))
		}

		var apiResp apiResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, err
		}
		if !apiResp.Success {
			return nil, fmt.Errorf("elcom api unsuccessful: %s", string(apiResp.Errors))
		}
		return apiResp.Data, nil
	}

	if lastErr == nil {
		lastErr = errors.New("elcom request failed")
	}
	return nil, lastErr
}

func isRetryableStatus(status int) bool {
	switch status {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func toProductRecord(raw map[string]any) (internal.ProductRecord, error) {
	header, _ := raw["header"].(string)
	header = strings.TrimSpace(header)
	if header == "" {
		return internal.ProductRecord{}, errors.New("empty header")
	}

	id, ok := toInt(raw["id"])
	if !ok {
		return internal.ProductRecord{}, errors.New("missing id")
	}

	rawJSON, _ := json.Marshal(raw)
	product := internal.ProductRecord{
		ID:      id,
		Header:  header,
		RawJSON: string(rawJSON),
	}
	product.SyncUID = toStringPtr(raw["syncUid"])
	product.Articul = toStringPtr(raw["articul"])
	product.UnitHeader = toStringPtr(raw["unitHeader"])
	product.ManufacturerHeader = toStringPtr(raw["manufacturerHeader"])
	product.MultiplicityOrder = toFloatPtr(raw["multiplicityOrder"])
	product.UpdatedAt = toStringPtr(raw["updatedAt"])
	product.FlatCodes = toFlatCodes(raw["flatCodes"])
	product.AnalogCodes = toStringSlice(raw["analogCodes"])

	return product, nil
}

func toInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case json.Number:
		i, err := t.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}

func toFloatPtr(v any) *float64 {
	switch t := v.(type) {
	case float64:
		return &t
	case int:
		f := float64(t)
		return &f
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return &f
		}
	}
	return nil
}

func toStringPtr(v any) *string {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return util.StringPtr(s)
}

func toStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func toFlatCodes(v any) internal.ProductFlatCodes {
	flat := internal.ProductFlatCodes{}
	m, ok := v.(map[string]any)
	if !ok {
		return flat
	}
	flat.Elcom = toStringPtr(m["elcom"])
	flat.Manufacturer = toStringPtr(m["manufacturer"])
	flat.Raec = toStringPtr(m["raec"])
	flat.PC = toStringPtr(m["pc"])
	flat.Etm = toStringPtr(m["etm"])
	return flat
}
