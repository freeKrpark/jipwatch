package ecos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeECOS(t *testing.T, body []byte) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	c := NewClient("testkey")
	c.Endpoint = srv.URL
	return c
}

func TestFetchM2(t *testing.T) {
	body := []byte(`{"StatisticSearch":{"list_total_count":2,"row":[
		{"STAT_CODE":"161Y006","ITEM_CODE1":"BBHA00","TIME":"202301","DATA_VALUE":"3812.5","UNIT_NAME":"조원"},
		{"STAT_CODE":"161Y006","ITEM_CODE1":"BBHA00","TIME":"202302","DATA_VALUE":"3830.1","UNIT_NAME":"조원"}
	]}}`)
	c := fakeECOS(t, body)
	rows, err := c.M2(context.Background(), "202301", "202302")
	if err != nil {
		t.Fatalf("M2: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].Value != "3812.5" || rows[1].Time != "202302" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestFetchBaseRate(t *testing.T) {
	body := []byte(`{"StatisticSearch":{"list_total_count":1,"row":[
		{"STAT_CODE":"722Y001","ITEM_CODE1":"0101000","TIME":"202301","DATA_VALUE":"3.50","UNIT_NAME":"연%"}
	]}}`)
	c := fakeECOS(t, body)
	rows, err := c.BaseRate(context.Background(), "202301", "202301")
	if err != nil {
		t.Fatalf("BaseRate: %v", err)
	}
	if len(rows) != 1 || rows[0].Value != "3.50" {
		t.Fatalf("unexpected: %+v", rows)
	}
}

func TestAPIError(t *testing.T) {
	body := []byte(`{"RESULT":{"CODE":"ERROR-100","MESSAGE":"인증키가 유효하지 않습니다."}}`)
	c := fakeECOS(t, body)
	err := c.Ping(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %v", err)
	}
	if apiErr.Code != "ERROR-100" {
		t.Fatalf("want ERROR-100, got %s", apiErr.Code)
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"3812.5", 3812.5, true},
		{"3.50", 3.5, true},
		{"1,234.5", 1234.5, true},
		{"-", 0, false},
		{"", 0, false},
		{"  ", 0, false},
	}
	for _, tt := range tests {
		v, ok := ParseValue(tt.in)
		if ok != tt.ok || (ok && v != tt.want) {
			t.Errorf("ParseValue(%q) = %v, %v; want %v, %v", tt.in, v, ok, tt.want, tt.ok)
		}
	}
}

func TestPing(t *testing.T) {
	body := []byte(`{"StatisticSearch":{"list_total_count":1,"row":[
		{"STAT_CODE":"722Y001","ITEM_CODE1":"0101000","TIME":"202301","DATA_VALUE":"3.50","UNIT_NAME":"연%"}
	]}}`)
	c := fakeECOS(t, body)
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
