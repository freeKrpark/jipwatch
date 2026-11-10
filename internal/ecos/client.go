package ecos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const DefaultEndpoint = "https://ecos.bok.or.kr/api/StatisticSearch"

// Client talks to the Bank of Korea ECOS Open API.
type Client struct {
	Key      string
	Endpoint string
	HTTP     *http.Client
}

func NewClient(key string) *Client {
	return &Client{
		Key:      strings.TrimSpace(key),
		Endpoint: DefaultEndpoint,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is a structured error returned by ECOS.
type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ecos api error %s: %s", e.Code, e.Message)
}

type Row struct {
	StatCode  string `json:"STAT_CODE"`
	ItemCode1 string `json:"ITEM_CODE1"`
	ItemName1 string `json:"ITEM_NAME1"`
	Time      string `json:"TIME"`
	Value     string `json:"DATA_VALUE"`
	Unit      string `json:"UNIT_NAME"`
}

func (c *Client) Fetch(ctx context.Context, statCode, cycle, startDate, endDate, itemCode1 string) ([]Row, error) {

	u := fmt.Sprintf("%s/%s/json/kr/1/10000/%s/%s/%s/%s",
		c.Endpoint, c.Key, statCode, cycle, startDate, endDate)
	if itemCode1 != "" {
		u += "/" + itemCode1
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ecos: HTTP %d: %.200q", resp.StatusCode, body)
	}
	return parseResponse(body)
}

func (c *Client) M2(ctx context.Context, startYM, endYM string) ([]Row, error) {
	return c.Fetch(ctx, "161Y006", "M", startYM, endYM, "BBHA00")
}

func (c *Client) BaseRate(ctx context.Context, startYM, endYM string) ([]Row, error) {
	return c.Fetch(ctx, "722Y001", "M", startYM, endYM, "0101000")
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Fetch(ctx, "722Y001", "M", "202301", "202301", "0101000")
	return err
}

func parseResponse(body []byte) ([]Row, error) {
	var errResp struct {
		Result struct {
			Code    string `json:"CODE"`
			Message string `json:"MESSAGE"`
		} `json:"RESULT"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Result.Code != "" {
		return nil, &APIError{Code: errResp.Result.Code, Message: errResp.Result.Message}
	}

	var resp struct {
		StatisticSearch struct {
			TotalCount int   `json:"list_total_count"`
			Row        []Row `json:"row"`
		} `json:"StatisticSearch"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("ecos: decode response: %w (%.200q)", err, body)
	}
	return resp.StatisticSearch.Row, nil
}

func ParseValue(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	return v, err == nil
}
