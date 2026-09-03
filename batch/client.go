package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultAPIKey = "example-ads-demo-key"

var errNotFound = errors.New("not found")

type Account struct {
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}

type accountsResponse struct {
	Data  []Account `json:"data"`
	Page  int       `json:"page"`
	Total int       `json:"total"`
}

type ReportRow struct {
	Date        string `json:"date"`
	Impressions int    `json:"impressions"`
	Clicks      int    `json:"clicks"`
	Cost        int    `json:"cost"`
	Conversions int    `json:"conversions"`
}

type reportsResponse struct {
	AccountID string      `json:"account_id"`
	Rows      []ReportRow `json:"rows"`
}

type ExampleAdsClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewExampleAdsClient(baseURL, apiKey string) *ExampleAdsClient {
	if apiKey == "" {
		apiKey = defaultAPIKey
	}
	return &ExampleAdsClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ExampleAdsClient) ListAccounts(ctx context.Context) ([]Account, error) {
	const limit = 100
	page := 1
	all := make([]Account, 0)

	for {
		var payload accountsResponse
		err := c.getJSON(ctx, "/v1/accounts", url.Values{
			"page":  {strconv.Itoa(page)},
			"limit": {strconv.Itoa(limit)},
		}, &payload)
		if err != nil {
			return nil, err
		}
		all = append(all, payload.Data...)
		if len(all) >= payload.Total || len(payload.Data) == 0 {
			break
		}
		page++
	}

	return all, nil
}

func (c *ExampleAdsClient) GetReports(ctx context.Context, accountID, dateFrom, dateTo string) ([]ReportRow, error) {
	var payload reportsResponse
	err := c.getJSON(ctx, "/v1/reports", url.Values{
		"account_id": {accountID},
		"date_from":  {dateFrom},
		"date_to":    {dateTo},
	}, &payload)
	if err != nil {
		return nil, err
	}
	return payload.Rows, nil
}

func (c *ExampleAdsClient) getJSON(ctx context.Context, path string, query url.Values, dest any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		res, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		switch res.StatusCode {
		case http.StatusOK:
			if err := json.Unmarshal(body, dest); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
			return nil
		case http.StatusNotFound:
			return errNotFound
		case http.StatusTooManyRequests:
			lastErr = fmt.Errorf("rate limited: %s", string(body))
			if retryAfter := res.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(time.Duration(seconds) * time.Second):
					}
				}
			}
			continue
		default:
			if res.StatusCode >= 500 {
				lastErr = fmt.Errorf("server error %d: %s", res.StatusCode, string(body))
				continue
			}
			return fmt.Errorf("api status %d: %s", res.StatusCode, string(body))
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("request failed after retries")
}

func isNotFound(err error) bool {
	return errors.Is(err, errNotFound)
}
