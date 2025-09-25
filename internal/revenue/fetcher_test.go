package revenue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

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
	result, err := svc.Fetch(ctx, "2330")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result.Records))
	}
	if result.Records[0].Month != 1 || result.Records[1].Month != 2 {
		t.Fatalf("records not sorted: %+v", result.Records)
	}
	if result.Source != SourceTWSE {
		t.Fatalf("expected source %s, got %s", SourceTWSE, result.Source)
	}
	if result.Company == nil {
		t.Fatalf("expected company metadata")

	}
}

func TestServiceFetchStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("bad"))
	}))
	defer server.Close()

	svc := &Service{Endpoint: server.URL}
	result, err := svc.Fetch(context.Background(), "2330")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Source != SourceFallback {
		t.Fatalf("expected fallback source, got %s", result.Source)
	}
	if result.Company == nil {
		t.Fatalf("expected company info in fallback")
	}
	if !strings.Contains(result.Note, "狀態碼") {
		t.Fatalf("expected note to mention status code: %s", result.Note)

	}
}

func TestServiceFetchNoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"公司代號":"1101","資料年月":"11201","營業收入-當月營收":"100"}]`))
	}))
	defer server.Close()

	svc := &Service{Endpoint: server.URL}
	_, err := svc.Fetch(context.Background(), "9999")

	if !errors.Is(err, ErrNoData) {
		t.Fatalf("expected ErrNoData, got %v", err)
	}
}

func TestServiceFetchFallbackWhenNoRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"公司代號":"1101","資料年月":"11201","營業收入-當月營收":"100"}]`))
	}))
	defer server.Close()

	svc := &Service{Endpoint: server.URL}
	result, err := svc.Fetch(context.Background(), "2330")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Source != SourceFallback {
		t.Fatalf("expected fallback source, got %s", result.Source)
	}
	if len(result.Records) == 0 {
		t.Fatalf("expected fallback records")
	}
}

