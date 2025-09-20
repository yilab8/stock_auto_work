package valuation

import (
	"math"
	"testing"
	"time"

	"github.com/yilab8/stock_auto_work/internal/revenue"
)

func TestBuildYearProjection(t *testing.T) {
	grouped := map[int][]revenue.MonthlyRevenue{
		2023: buildMonthlyRevenue(2023, []float64{300, 280, 320, 330, 340, 350, 360, 370, 380, 390, 400, 410}),
		2024: {
			{Year: 2024, Month: time.January, Revenue: 388},
			{Year: 2024, Month: time.February, Revenue: 388},
		},
	}
	manual := map[time.Month]float64{
		time.March: 0.1,
	}
	asm := Assumptions{
		GrossMargin:        0.173,
		OperatingExpense:   38,
		NonOperatingIncome: 38,
		TaxRate:            0.2,
		SharesOutstanding:  80,
		PerMultiple:        23,
		CurrentPrice:       56,
	}
	projection, err := BuildYearProjection(2024, grouped, manual, asm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projection.Months) != 12 {
		t.Fatalf("expected 12 months, got %d", len(projection.Months))
	}
	jan := projection.Months[0]
	if !jan.IsActual {
		t.Fatalf("January should be actual")
	}
	expectedJanYoY := (388.0 - 300.0) / 300.0
	if math.Abs(jan.YoY-expectedJanYoY) > 1e-6 {
		t.Fatalf("unexpected Jan YoY: %f", jan.YoY)
	}
	mar := projection.Months[2]
	if mar.IsActual {
		t.Fatalf("March should be estimated")
	}
	if math.Abs(mar.Revenue-352.0) > 1e-6 {
		t.Fatalf("unexpected March revenue: %f", mar.Revenue)
	}
	q1 := projection.Quarters[0]
	if q1.Quarter != 1 {
		t.Fatalf("unexpected quarter number: %d", q1.Quarter)
	}
	expectedQ1Revenue := 388.0 + 388.0 + 352.0
	if math.Abs(q1.Revenue-expectedQ1Revenue) > 1e-6 {
		t.Fatalf("unexpected Q1 revenue: %f", q1.Revenue)
	}
	expectedQ1EPS := ((expectedQ1Revenue*0.173 - 38.0) + 38.0) * (1 - 0.2) / 80.0
	if math.Abs(q1.EPS-expectedQ1EPS) > 1e-6 {
		t.Fatalf("unexpected Q1 EPS: %f", q1.EPS)
	}
	if projection.EstimatedPrice <= 0 {
		t.Fatalf("expected positive estimated price")
	}
	if projection.AvgYoY <= 0 {
		t.Fatalf("expected positive avg YoY")
	}
}

func TestBuildYearProjectionSharesError(t *testing.T) {
	grouped := map[int][]revenue.MonthlyRevenue{
		2023: buildMonthlyRevenue(2023, []float64{100, 100, 100}),
		2024: {{Year: 2024, Month: time.January, Revenue: 200}},
	}
	asm := Assumptions{SharesOutstanding: 0.0}
	_, err := BuildYearProjection(2024, grouped, nil, asm)
	if err == nil {
		t.Fatalf("expected error for zero shares")
	}
}

func buildMonthlyRevenue(year int, values []float64) []revenue.MonthlyRevenue {
	out := make([]revenue.MonthlyRevenue, len(values))
	for i, val := range values {
		out[i] = revenue.MonthlyRevenue{Year: year, Month: time.Month(i + 1), Revenue: val}
	}
	return out
}
