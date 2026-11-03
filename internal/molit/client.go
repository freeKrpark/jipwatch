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

const (
	DefaultEndpoint = "https://apis.data.go.kr/1613000/RTMSDataSvcAptTradeDev/getRTMSDataSvcAptTradeDev"
	RentEndpoint    = "https://apis.data.go.kr/1613000/RTMSDataSvcAptRent/getRTMSDataSvcAptRent"
)

type Client struct {
	Key          string
	Endpoint     string // sales
	RentEndpoint string // 전월세
	PageSize     int
	HTTP         *http.Client
}

func NewClient(key string) *Client {
	return &Client{
		Key:          NormalizeKey(key),
		Endpoint:     DefaultEndpoint,
		RentEndpoint: RentEndpoint,
		PageSize:     1000,
		HTTP:         &http.Client{Timeout: 30 * time.Second},
	}
}

// Month fetches every sale in district sggCd in month ym.
func (c *Client) Month(ctx context.Context, sggCd, ym string) ([]Trade, error) {
	return fetchMonth(ctx, c, c.Endpoint, sggCd, ym, Parse)
}

// Rents fetches every 전월세 contract in district sggCd in month ym. The
// rent API needs its own 활용신청; without it this fails with APIError 30.
func (c *Client) Rents(ctx context.Context, sggCd, ym string) ([]Rent, error) {
	return fetchMonth(ctx, c, c.RentEndpoint, sggCd, ym, ParseRents)
}

func fetchMonth[T any](ctx context.Context, c *Client, endpoint, sggCd, ym string,
	parse func([]byte) ([]T, int, error)) ([]T, error) {
	var all []T
	for page := 1; ; page++ {
		rows, total, err := fetchPage(ctx, c, endpoint, sggCd, ym, page, c.PageSize, parse)
		if err != nil {
			return nil, fmt.Errorf("%s %s page %d: %w", sggCd, ym, page, err)
		}
		all = append(all, rows...)
		if len(all) >= total {
			return all, nil
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("%s %s: got %d of %d rows", sggCd, ym, len(all), total)
		}
	}
}

func (c *Client) Ping(ctx context.Context) error {
	_, _, err := fetchPage(ctx, c, c.Endpoint, "11110", "202301", 1, 1, Parse)
	return err
}

func fetchPage[T any](ctx context.Context, c *Client, endpoint, sggCd, ym string, page, rows int,
	parse func([]byte) ([]T, int, error)) ([]T, int, error) {
	q := url.Values{
		"serviceKey": {c.Key},
		"LAWD_CD":    {sggCd},
		"DEAL_YMD":   {ym},
		"pageNo":     {strconv.Itoa(page)},
		"numOfRows":  {strconv.Itoa(rows)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
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
		if _, _, err := parse(body); errors.As(err, &apiErr) {
			return nil, 0, apiErr
		}
		return nil, 0, fmt.Errorf("HTTP %d: %.120q", resp.StatusCode, body)
	}
	return parse(body)
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
