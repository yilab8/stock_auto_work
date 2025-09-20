package server

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/yilab8/stock_auto_work/internal/revenue"
)

type stubFetcher struct {
	records []revenue.MonthlyRevenue
	err     error
}

func (s *stubFetcher) Fetch(ctx context.Context, stockNo string) ([]revenue.MonthlyRevenue, error) {
	return s.records, s.err
}

func TestParseYoYInputs(t *testing.T) {
	values := url.Values{}
	values.Set("yoy_01", "10")
	values.Set("yoy_02", "")
	values.Set("yoy_03", "-5")
	result := parseYoYInputs(values)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[time.January] != 0.1 {
		t.Fatalf("unexpected value: %f", result[time.January])
	}
	if result[time.March] != -0.05 {
		t.Fatalf("unexpected value: %f", result[time.March])
	}
}

func TestAppServeHTTP(t *testing.T) {
	records := make([]revenue.MonthlyRevenue, 0, 16)
	base2023 := []float64{300, 280, 320, 330, 340, 350, 360, 370, 380, 390, 400, 410}
	for i, v := range base2023 {
		records = append(records, revenue.MonthlyRevenue{Year: 2023, Month: time.Month(i + 1), Revenue: v})
	}
	records = append(records, revenue.MonthlyRevenue{Year: 2024, Month: time.January, Revenue: 388})

	tmpl := template.Must(template.New("test").Parse(`{{(index .Months 0).Label}}|{{printf "%.0f" (index .Months 1).Revenue}}|{{printf "%.2f" .Summary.AnnualEPS}}`))
	app := NewApp(&stubFetcher{records: records}, tmpl)
	app.now = func() time.Time { return time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC) }

	req := httptest.NewRequest(http.MethodGet, "/?stock_no=2330&yoy_02=10", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "01月") {
		t.Fatalf("missing month label: %s", body)
	}
	if !strings.Contains(body, "308") {
		t.Fatalf("manual yoy not applied: %s", body)
	}
}
