package logquery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestClient_QueryAndCount(t *testing.T) {
	var gotPath, gotKey, gotRaw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotKey, gotRaw = r.URL.Path, r.Header.Get("X-API-Key"), r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"count":1}}`))
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL + "/api/v1/", APIKey: "k"})
	params := url.Values{"service": {"sereni-base"}, "fields.base_id": {"bs_abc"}}

	res, err := c.Query(context.Background(), params)
	if err != nil || res.Status != 200 || string(res.Body) != `{"success":true,"data":{"count":1}}` {
		t.Fatalf("query: %v %+v", err, res)
	}
	if gotPath != "/api/v1/logs/query" || gotKey != "k" {
		t.Fatalf("path %q key %q", gotPath, gotKey)
	}
	if q, _ := url.ParseQuery(gotRaw); q.Get("fields.base_id") != "bs_abc" || q.Get("service") != "sereni-base" {
		t.Fatalf("params not sent: %s", gotRaw)
	}

	if _, err = c.Count(context.Background(), params); err != nil || gotPath != "/api/v1/logs/count" {
		t.Fatalf("count: %v path %q", err, gotPath)
	}
}

func TestClient_NoKeyHeaderWhenUnset(t *testing.T) {
	var hasKey bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hasKey = r.Header["X-Api-Key"]
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	if _, err := New(Config{URL: srv.URL}).Query(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if hasKey {
		t.Fatal("X-API-Key must not be sent when no key is set")
	}
}

func TestClient_StatusHandling(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr error
	}{
		{"200 passes", 200, nil},
		{"400 passes as it is", 400, nil},
		{"401 is our key failing, not the caller's login", 401, ErrUnavailable},
		{"502", 502, ErrUnavailable},
		{"500", 500, ErrUnavailable},
		{"404", 404, ErrUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"message":"from log service"}`))
			}))
			defer srv.Close()
			res, err := New(Config{URL: srv.URL}).Query(context.Background(), nil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (res.Status != tt.status || string(res.Body) != `{"message":"from log service"}`) {
				t.Fatalf("answer changed: %+v", res)
			}
		})
	}
}

func TestClient_NotConfigured(t *testing.T) {
	if _, err := New(Config{}).Query(context.Background(), nil); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v", err)
	}
	var nilClient *Client
	if _, err := nilClient.Count(context.Background(), nil); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("nil client err = %v", err)
	}
}

func TestClient_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()
	_, err := New(Config{URL: srv.URL, Timeout: 50 * time.Millisecond}).Query(context.Background(), nil)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want timeout", err)
	}
}

func TestClient_Unreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	_, err := New(Config{URL: url}).Query(context.Background(), nil)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want unavailable", err)
	}
}
