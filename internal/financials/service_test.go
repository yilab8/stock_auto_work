package financials

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServiceFetch(t *testing.T) {
	payload := []RawQuarterRecord{
		{
			"公司代號":   "2330",
			"年度":     "2024",
			"季別":     "2",
			"稅後淨利":   "236327000",
			"基本每股盈餘": "9.0",
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	svc := Service{Endpoint: srv.URL}
	result, err := svc.Fetch(context.Background(), "2330")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("unexpected record length: %d", len(result.Records))
	}
	if result.Records[0].BasicEPS != 9.0 {
		t.Fatalf("unexpected eps: %f", result.Records[0].BasicEPS)
	}
}

func TestServiceFallback(t *testing.T) {
	svc := Service{Endpoint: "http://127.0.0.1:65535", Client: &http.Client{Timeout: 50 * time.Millisecond}}
	result, err := svc.Fetch(context.Background(), "2317")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatalf("expected fallback records")
	}
	if result.Source != SourceFallback {
		t.Fatalf("unexpected source: %s", result.Source)
	}
}
