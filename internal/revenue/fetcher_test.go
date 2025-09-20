package revenue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServiceFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
            {"公司代號":"2330","資料年月":"11201","營業收入-當月營收":"100"},
            {"公司代號":"1101","資料年月":"11201","營業收入-當月營收":"200"},
            {"公司代號":"2330","資料年月":"11202","營業收入-當月營收":"300"}
        ]`))
	}))
	defer server.Close()

	svc := &Service{Endpoint: server.URL}
	ctx := context.Background()
	records, err := svc.Fetch(ctx, "2330")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Month != 1 || records[1].Month != 2 {
		t.Fatalf("records not sorted: %+v", records)
	}
}

func TestServiceFetchStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("bad"))
	}))
	defer server.Close()

	svc := &Service{Endpoint: server.URL}
	_, err := svc.Fetch(context.Background(), "2330")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestServiceFetchNoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"公司代號":"1101","資料年月":"11201","營業收入-當月營收":"100"}]`))
	}))
	defer server.Close()

	svc := &Service{Endpoint: server.URL}
	_, err := svc.Fetch(context.Background(), "2330")
	if !errors.Is(err, ErrNoData) {
		t.Fatalf("expected ErrNoData, got %v", err)
	}
}
