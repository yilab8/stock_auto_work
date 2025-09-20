package valuation

import (
	"fmt"
	"sort"
	"time"

	"github.com/yilab8/stock_auto_work/internal/revenue"
)

// MonthEstimate 描述單月營收與年增率狀態。
type MonthEstimate struct {
	Year            int
	Month           time.Month
	Revenue         float64
	PreviousRevenue float64
	YoY             float64 // 例如 0.15 表示 15%
	IsActual        bool
}

// QuarterInputs 為單季計算 EPS 所需的基本假設。
type QuarterInputs struct {
	GrossMargin        float64
	OperatingExpense   float64
	NonOperatingIncome float64
	TaxRate            float64
}

// QuarterBreakdown 保留單季重要推估數據。
type QuarterBreakdown struct {
	Quarter         int
	Revenue         float64
	GrossProfit     float64
	OperatingIncome float64
	PreTaxIncome    float64
	NetIncome       float64
	EPS             float64
	IsActual        bool
}

// Assumptions 為全年估值使用的主要輸入值。
type Assumptions struct {
	GrossMargin        float64
	OperatingExpense   float64
	NonOperatingIncome float64
	TaxRate            float64
	SharesOutstanding  float64
	PrevQuarterEPS     float64
	PerMultiple        float64
	CurrentPrice       float64
	QuarterOverrides   map[int]QuarterOverride
}

// QuarterOverride 提供個別季度的覆寫參數。
type QuarterOverride struct {
	GrossMargin        *float64
	OperatingExpense   *float64
	NonOperatingIncome *float64
	TaxRate            *float64
}

// YearProjection 結果彙總。
type YearProjection struct {
	Year           int
	Months         []MonthEstimate
	Quarters       []QuarterBreakdown
	AnnualRevenue  float64
	AnnualEPS      float64
	EstimatedPrice float64
	Upside         float64
	AvgYoY         float64
}

// BuildYearProjection 組合單一年份的營收推估與估值計算。
func BuildYearProjection(year int, grouped map[int][]revenue.MonthlyRevenue, manualYoY map[time.Month]float64, asm Assumptions) (YearProjection, error) {
	current := grouped[year]
	previous := grouped[year-1]
	if len(current) == 0 {
		return YearProjection{}, fmt.Errorf("缺少 %d 年的營收資料", year)
	}
	if asm.SharesOutstanding <= 0 {
		return YearProjection{}, fmt.Errorf("SharesOutstanding 必須大於 0")
	}

	monthMap := make(map[time.Month]revenue.MonthlyRevenue)
	for _, rec := range current {
		if rec.Year != year {
			continue
		}
		monthMap[rec.Month] = rec
	}
	prevMap := make(map[time.Month]float64)
	for _, rec := range previous {
		prevMap[rec.Month] = rec.Revenue
	}

	yoySum := 0.0
	yoyCount := 0
	actualYoY := make(map[time.Month]float64)
	for month, rec := range monthMap {
		prev := prevMap[month]
		if prev <= 0 {
			continue
		}
		yoy := (rec.Revenue - prev) / prev
		actualYoY[month] = yoy
		yoySum += yoy
		yoyCount++
	}
	avgYoY := 0.0
	if yoyCount > 0 {
		avgYoY = yoySum / float64(yoyCount)
	}

	months := make([]MonthEstimate, 0, 12)
	totalRevenue := 0.0
	for m := time.January; m <= time.December; m++ {
		if rec, ok := monthMap[m]; ok {
			yoy := actualYoY[m]
			prev := prevMap[m]
			months = append(months, MonthEstimate{
				Year:            year,
				Month:           m,
				Revenue:         rec.Revenue,
				PreviousRevenue: prev,
				YoY:             yoy,
				IsActual:        true,
			})
			totalRevenue += rec.Revenue
			continue
		}
		prev := prevMap[m]
		yoy := avgYoY
		if v, ok := manualYoY[m]; ok {
			yoy = v
		}
		revenue := prev * (1 + yoy)
		months = append(months, MonthEstimate{
			Year:            year,
			Month:           m,
			Revenue:         revenue,
			PreviousRevenue: prev,
			YoY:             yoy,
			IsActual:        false,
		})
		totalRevenue += revenue
	}
	sort.Slice(months, func(i, j int) bool { return months[i].Month < months[j].Month })

	quarters := make([]QuarterBreakdown, 0, 4)
	annualEPS := 0.0
	for q := 1; q <= 4; q++ {
		start := (q-1)*3 + 1
		end := start + 2
		revenueSum := 0.0
		allActual := true
		for m := start; m <= end; m++ {
			month := time.Month(m)
			for _, item := range months {
				if item.Month == month {
					revenueSum += item.Revenue
					if !item.IsActual {
						allActual = false
					}
					break
				}
			}
		}
		inputs := asm.quarterInputs(q)
		gross := revenueSum * inputs.GrossMargin
		operating := gross - inputs.OperatingExpense
		preTax := operating + inputs.NonOperatingIncome
		netIncome := preTax * (1 - inputs.TaxRate)
		eps := netIncome / asm.SharesOutstanding
		quarter := QuarterBreakdown{
			Quarter:         q,
			Revenue:         revenueSum,
			GrossProfit:     gross,
			OperatingIncome: operating,
			PreTaxIncome:    preTax,
			NetIncome:       netIncome,
			EPS:             eps,
			IsActual:        allActual,
		}
		quarters = append(quarters, quarter)
		annualEPS += eps
	}

	estimatedPrice := annualEPS * asm.PerMultiple
	upside := 0.0
	if asm.CurrentPrice > 0 {
		upside = (estimatedPrice - asm.CurrentPrice) / asm.CurrentPrice
	}

	return YearProjection{
		Year:           year,
		Months:         months,
		Quarters:       quarters,
		AnnualRevenue:  totalRevenue,
		AnnualEPS:      annualEPS,
		EstimatedPrice: estimatedPrice,
		Upside:         upside,
		AvgYoY:         avgYoY,
	}, nil
}

func (a Assumptions) quarterInputs(q int) QuarterInputs {
	inputs := QuarterInputs{
		GrossMargin:        a.GrossMargin,
		OperatingExpense:   a.OperatingExpense,
		NonOperatingIncome: a.NonOperatingIncome,
		TaxRate:            a.TaxRate,
	}
	if a.QuarterOverrides == nil {
		return inputs
	}
	if override, ok := a.QuarterOverrides[q]; ok {
		if override.GrossMargin != nil {
			inputs.GrossMargin = *override.GrossMargin
		}
		if override.OperatingExpense != nil {
			inputs.OperatingExpense = *override.OperatingExpense
		}
		if override.NonOperatingIncome != nil {
			inputs.NonOperatingIncome = *override.NonOperatingIncome
		}
		if override.TaxRate != nil {
			inputs.TaxRate = *override.TaxRate
		}
	}
	return inputs
}
