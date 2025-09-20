package server

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/yilab8/stock_auto_work/internal/revenue"
	"github.com/yilab8/stock_auto_work/internal/valuation"
)

// RevenueFetcher 抽象化營收服務，方便測試替換。
type RevenueFetcher interface {
	Fetch(ctx context.Context, stockNo string) ([]revenue.MonthlyRevenue, error)
}

// App 提供網站主要服務。
type App struct {
	Fetcher  RevenueFetcher
	Template *template.Template
	now      func() time.Time
}

// NewApp 建立 App 實例。
func NewApp(fetcher RevenueFetcher, tmpl *template.Template) *App {
	return &App{
		Fetcher:  fetcher,
		Template: tmpl,
		now:      time.Now,
	}
}

// ServeHTTP 處理首頁請求。
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.Template == nil {
		http.Error(w, "template not configured", http.StatusInternalServerError)
		return
	}
	query := r.URL.Query()
	stockNo := query.Get("stock_no")
	if stockNo == "" {
		stockNo = "2330"
	}
	year := parseInt(query, "year", a.now().Year())
	form := buildForm(query)

	data := &pageData{
		StockNo: stockNo,
		Year:    year,
		Form:    form,
	}

	if a.Fetcher == nil {
		data.Error = "尚未設定資料來源"
		a.render(w, data)
		return
	}

	records, err := a.Fetcher.Fetch(r.Context(), stockNo)
	if err != nil {
		data.Error = fmt.Sprintf("取得營收資料失敗: %v", err)
		a.render(w, data)
		return
	}

	grouped := revenue.GroupByYear(records)
	years := make([]int, 0, len(grouped))
	for y := range grouped {
		years = append(years, y)
	}
	sort.Ints(years)
	data.AvailableYears = years

	manualYoY := parseYoYInputs(query)
	projection, err := valuation.BuildYearProjection(year, grouped, manualYoY, form.toAssumptions())
	if err != nil {
		data.Error = err.Error()
		a.render(w, data)
		return
	}

	data.Projection = &projection
	data.Months = buildMonthViews(projection.Months, manualYoY)
	data.Quarters = buildQuarterViews(projection.Quarters)
	data.Summary = buildSummary(projection)
	a.render(w, data)
}

func (a *App) render(w http.ResponseWriter, data *pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type pageData struct {
	StockNo        string
	Year           int
	AvailableYears []int
	Form           formValues
	Projection     *valuation.YearProjection
	Months         []monthView
	Quarters       []quarterView
	Summary        summaryView
	Error          string
}

type formValues struct {
	GrossMargin        string
	OperatingExpense   string
	NonOperatingIncome string
	TaxRate            string
	Shares             string
	PrevQuarterEPS     string
	PerMultiple        string
	CurrentPrice       string
}

func buildForm(query url.Values) formValues {
	return formValues{
		GrossMargin:        defaultString(query, "gross_margin", "17.3"),
		OperatingExpense:   defaultString(query, "operating_expense", "38"),
		NonOperatingIncome: defaultString(query, "non_operating_income", "38"),
		TaxRate:            defaultString(query, "tax_rate", "20"),
		Shares:             defaultString(query, "shares", "80"),
		PrevQuarterEPS:     defaultString(query, "prev_eps", "0.8"),
		PerMultiple:        defaultString(query, "per", "23"),
		CurrentPrice:       defaultString(query, "current_price", "56"),
	}
}

func (f formValues) toAssumptions() valuation.Assumptions {
	grossMargin := parsePercentString(f.GrossMargin)
	taxRate := parsePercentString(f.TaxRate)
	operatingExpense := parseFloatString(f.OperatingExpense)
	nonOperating := parseFloatString(f.NonOperatingIncome)
	shares := parseFloatString(f.Shares)
	prevEPS := parseFloatString(f.PrevQuarterEPS)
	per := parseFloatString(f.PerMultiple)
	price := parseFloatString(f.CurrentPrice)

	return valuation.Assumptions{
		GrossMargin:        grossMargin,
		OperatingExpense:   operatingExpense,
		NonOperatingIncome: nonOperating,
		TaxRate:            taxRate,
		SharesOutstanding:  shares,
		PrevQuarterEPS:     prevEPS,
		PerMultiple:        per,
		CurrentPrice:       price,
	}
}

type monthView struct {
	Index           int
	Label           string
	PreviousRevenue float64
	Revenue         float64
	YoYPercent      float64
	IsActual        bool
	InputName       string
	InputValue      string
	Editable        bool
}

type quarterView struct {
	Quarter         int
	Revenue         float64
	GrossProfit     float64
	OperatingIncome float64
	PreTaxIncome    float64
	NetIncome       float64
	EPS             float64
	IsActual        bool
}

type summaryView struct {
	AnnualRevenue  float64
	AnnualEPS      float64
	EstimatedPrice float64
	UpsidePercent  float64
	AvgYoYPercent  float64
}

func buildMonthViews(months []valuation.MonthEstimate, manual map[time.Month]float64) []monthView {
	views := make([]monthView, 0, len(months))
	for _, m := range months {
		name := fmt.Sprintf("yoy_%02d", int(m.Month))
		inputValue := ""
		if v, ok := manual[m.Month]; ok {
			inputValue = fmt.Sprintf("%.2f", v*100)
		} else {
			inputValue = fmt.Sprintf("%.2f", m.YoY*100)
		}
		views = append(views, monthView{
			Index:           int(m.Month),
			Label:           fmt.Sprintf("%02d月", int(m.Month)),
			PreviousRevenue: m.PreviousRevenue,
			Revenue:         m.Revenue,
			YoYPercent:      m.YoY * 100,
			IsActual:        m.IsActual,
			InputName:       name,
			InputValue:      inputValue,
			Editable:        !m.IsActual,
		})
	}
	return views
}

func buildQuarterViews(quarters []valuation.QuarterBreakdown) []quarterView {
	views := make([]quarterView, 0, len(quarters))
	for _, q := range quarters {
		views = append(views, quarterView{
			Quarter:         q.Quarter,
			Revenue:         q.Revenue,
			GrossProfit:     q.GrossProfit,
			OperatingIncome: q.OperatingIncome,
			PreTaxIncome:    q.PreTaxIncome,
			NetIncome:       q.NetIncome,
			EPS:             q.EPS,
			IsActual:        q.IsActual,
		})
	}
	return views
}

func buildSummary(p valuation.YearProjection) summaryView {
	upsidePercent := p.Upside * 100
	avgYoYPercent := p.AvgYoY * 100
	return summaryView{
		AnnualRevenue:  p.AnnualRevenue,
		AnnualEPS:      p.AnnualEPS,
		EstimatedPrice: p.EstimatedPrice,
		UpsidePercent:  upsidePercent,
		AvgYoYPercent:  avgYoYPercent,
	}
}

func parseInt(values url.Values, key string, def int) int {
	raw := values.Get(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func defaultString(values url.Values, key, def string) string {
	if raw := values.Get(key); raw != "" {
		return raw
	}
	return def
}

func parsePercentString(raw string) float64 {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v / 100
}

func parseFloatString(raw string) float64 {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseYoYInputs(values url.Values) map[time.Month]float64 {
	result := make(map[time.Month]float64)
	for m := 1; m <= 12; m++ {
		key := fmt.Sprintf("yoy_%02d", m)
		raw := values.Get(key)
		if raw == "" {
			continue
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		result[time.Month(m)] = v / 100
	}
	return result
}
