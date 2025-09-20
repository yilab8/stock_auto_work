package revenue

import (
	"testing"
	"time"
)

func TestParseYearMonthROC(t *testing.T) {
	rec := RawRecord{DataMonth: "11205"}
	year, month, err := rec.ParseYearMonth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if year != 2023 || month != time.May {
		t.Fatalf("unexpected result: %d %v", year, month)
	}
}

func TestParseYearMonthAD(t *testing.T) {
	rec := RawRecord{DataMonth: "202312"}
	year, month, err := rec.ParseYearMonth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if year != 2023 || month != time.December {
		t.Fatalf("unexpected result: %d %v", year, month)
	}
}

func TestParseRevenue(t *testing.T) {
	rec := RawRecord{MonthlyRevenue: "1,234,567"}
	val, err := rec.ParseRevenue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 1234567 {
		t.Fatalf("unexpected value: %f", val)
	}
}

func TestNormalize(t *testing.T) {
	rec := RawRecord{DataMonth: "11201", MonthlyRevenue: "10,000"}
	m, err := rec.Normalize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Year != 2023 || m.Month != time.January || m.Revenue != 10000 {
		t.Fatalf("unexpected normalize result: %+v", m)
	}
}

func TestFilterByStock(t *testing.T) {
	recs := []RawRecord{{CompanyCode: "2330"}, {CompanyCode: "2303"}, {CompanyCode: "2330"}}
	out := FilterByStock(recs, "2330")
	if len(out) != 2 {
		t.Fatalf("expected 2 records, got %d", len(out))
	}
}

func TestGroupByYear(t *testing.T) {
	recs := []MonthlyRevenue{{Year: 2022}, {Year: 2023}, {Year: 2022}}
	grouped := GroupByYear(recs)
	if len(grouped[2022]) != 2 || len(grouped[2023]) != 1 {
		t.Fatalf("unexpected grouping result: %+v", grouped)
	}
}

func TestSortMonthlyRevenues(t *testing.T) {
	recs := []MonthlyRevenue{{Year: 2023, Month: time.March}, {Year: 2022, Month: time.May}, {Year: 2023, Month: time.January}}
	sorted := SortMonthlyRevenues(recs)
	if sorted[0].Year != 2022 || sorted[1].Month != time.January {
		t.Fatalf("unexpected sort order: %+v", sorted)
	}
}
