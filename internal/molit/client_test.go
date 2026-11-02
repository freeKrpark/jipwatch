package molit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func fakeAPI(t *testing.T, total, pageSize int) (*httptest.Server, *[]string) {
	var pages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("serviceKey") != "key+/=" || q.Get("LAWD_CD") != "11110" || q.Get("DEAL_YMD") != "202304" {
			t.Errorf("unexpected query %v", q)
		}
		pages = append(pages, q.Get("pageNo"))
		n, _ := strconv.Atoi(q.Get("pageNo"))
		var rows []string
		for i := (n - 1) * pageSize; i < n*pageSize && i < total; i++ {
			rows = append(rows, row)
		}
		w.Write(page(strconv.Itoa(total), rows...))
	}))
	t.Cleanup(srv.Close)
	return srv, &pages
}

func testClient(url string, pageSize int) *Client {
	c := NewClient("key+/=") // characters that must be escaped in a query
	c.Endpoint, c.PageSize = url, pageSize
	return c
}

func TestMonthPages(t *testing.T) {
	srv, pages := fakeAPI(t, 5, 2)
	trades, err := testClient(srv.URL, 2).Month(context.Background(), "11110", "202304")
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 5 || len(*pages) != 3 {
		t.Fatalf("got %d trades in %d pages, want 5 in 3", len(trades), len(*pages))
	}
}

func TestMonthEmpty(t *testing.T) {
	srv, pages := fakeAPI(t, 0, 2)
	trades, err := testClient(srv.URL, 2).Month(context.Background(), "11110", "202304")
	if err != nil || len(trades) != 0 || len(*pages) != 1 {
		t.Fatalf("got %d trades in %d pages, err %v", len(trades), len(*pages), err)
	}
}

// A total the pages never reach must end in an error, not a loop.
func TestMonthShortPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pageNo") == "1" {
			w.Write(page("3", row))
			return
		}
		w.Write(page("3"))
	}))
	defer srv.Close()
	if _, err := testClient(srv.URL, 1).Month(context.Background(), "11110", "202304"); err == nil {
		t.Fatal("want error")
	}
}

func TestMonthHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream down", http.StatusBadGateway)
	}))
	defer srv.Close()
	if _, err := testClient(srv.URL, 10).Month(context.Background(), "11110", "202304"); err == nil {
		t.Fatal("want error")
	}
}

func TestMonthAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(keyError))
	}))
	defer srv.Close()
	_, err := testClient(srv.URL, 10).Month(context.Background(), "11110", "202304")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "30" {
		t.Fatalf("want APIError 30 through the wrapping, got %v", err)
	}
}
