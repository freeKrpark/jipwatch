package molit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultEndpoint = "https://apis.data.go.kr/1613000/RTMSDataSvcAptTradeDev/getRTMSDataSvcAptTradeDev"

type Client struct {
	Key      string
	Endpoint string
	PageSize int
	HTTP     *http.Client
}

func NewClient(key string) *Client {
	return &Client{
		Key:      NormalizeKey(key),
		Endpoint: DefaultEndpoint,
		PageSize: 1000,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Month(ctx context.Context, sggCd, ym string) ([]Trade, error) {
	var all []Trade
	for page := 1; ; page++ {
		trades, total, err := c.page(ctx, sggCd, ym, page, c.PageSize)
		if err != nil {
			return nil, fmt.Errorf("%s %s page %d: %w", sggCd, ym, page, err)
		}
		all = append(all, trades...)
		if len(all) >= total {
			return all, nil
		}
		if len(trades) == 0 {
			return nil, fmt.Errorf("%s %s: got %d of %d trades", sggCd, ym, len(all), total)
		}

	}
}

func (c *Client) Ping(ctx context.Context) error {
	_, _, err := c.page(ctx, "11110", "202301", 1, 1)
	return err
}

func (c *Client) page(ctx context.Context, sggCd, ym string, page, rows int) ([]Trade, int, error) {
	q := url.Values{
		"serviceKey": {c.Key},
		"LAWD_CD":    {sggCd},
		"DEAL_YMD":   {ym},
		"pageNo":     {strconv.Itoa(page)},
		"numOfRows":  {strconv.Itoa(rows)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr *APIError
		if _, _, err := Parse(body); errors.As(err, &apiErr) {
			return nil, 0, apiErr
		}
		return nil, 0, fmt.Errorf("HTTP %d: %.120q", resp.StatusCode, body)
	}
	return Parse(body)
}

func NormalizeKey(key string) string {
	key = strings.TrimSpace(key)
	if !strings.Contains(key, "%") {
		return key
	}
	if decoded, err := url.PathUnescape(key); err == nil {
		return decoded
	}
	return key
}
