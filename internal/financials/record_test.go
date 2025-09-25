package financials

import "testing"

func TestNormalize(t *testing.T) {
	raw := RawQuarterRecord{
		"公司代號":      "2330",
		"年度":        "2024",
		"季別":        "Q2",
		"稅後淨利":      "236,327,000",
		"基本每股盈餘(元)": "9.00",
	}
	report, err := raw.Normalize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Year != 2024 || report.Quarter != 2 {
		t.Fatalf("unexpected period: %+v", report)
	}
	if report.NetIncome != 236327000 {
		t.Fatalf("unexpected net income: %f", report.NetIncome)
	}
	if report.BasicEPS != 9.0 {
		t.Fatalf("unexpected eps: %f", report.BasicEPS)
	}
}

func TestParseQuarterInvalid(t *testing.T) {
	if _, err := parseQuarter("Q5"); err == nil {
		t.Fatalf("expected error for invalid quarter")
	}
}
