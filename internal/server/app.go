package server

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yilab8/stock_auto_work/internal/financials"
	"github.com/yilab8/stock_auto_work/internal/revenue"
	"github.com/yilab8/stock_auto_work/internal/valuation"
)

// RevenueFetcher 抽象化營收服務，方便測試替換。
type RevenueFetcher interface {
	Fetch(ctx context.Context, stockNo string) (revenue.FetchResult, error)
}

// EarningsFetcher 抽象化檢表稅後淨利資料來源。
type EarningsFetcher interface {
	Fetch(ctx context.Context, stockNo string) (financials.FetchResult, error)
}

// App 提供網站主要服務。
type App struct {
	Fetcher  RevenueFetcher
	Earnings EarningsFetcher
	Template *template.Template
	now      func() time.Time
}

// NewApp 建立 App 實例。
func NewApp(fetcher RevenueFetcher, earnings EarningsFetcher, tmpl *template.Template) *App {
	return &App{
		Fetcher:  fetcher,
		Earnings: earnings,
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
	stockNo := strings.TrimSpace(query.Get("stock_no"))
	if stockNo == "" {
		stockNo = "2330"
	}
	baseCompany, _ := revenue.LookupStaticCompany(stockNo)
	form := buildForm(query, baseCompany)

	data := &pageData{
		StockNo:      stockNo,
		Year:         a.now().Year(),
		Form:         form,
		Company:      toCompanyView(baseCompany),
		StockOptions: buildStockOptions(),
	}

	if a.Fetcher == nil {
		data.Error = "尚未設定資料來源"
		a.render(w, data)
		return
	}

	result, err := a.Fetcher.Fetch(r.Context(), stockNo)
	if err != nil {
		data.Error = fmt.Sprintf("取得營收資料失敗: %v", err)
		a.render(w, data)
		return
	}
	if len(result.Records) == 0 {
		data.Error = "取得的營收資料為空"
		a.render(w, data)
		return
	}
	if result.Company != nil {
		data.Company = toCompanyView(result.Company)
		form = buildForm(query, result.Company)
		data.Form = form
	}
	data.DataSource = result.Source
	data.DataNote = result.Note

	var earningsResult financials.FetchResult
	if a.Earnings != nil {
		if res, err := a.Earnings.Fetch(r.Context(), stockNo); err != nil {
			data.EarningsNote = fmt.Sprintf("取得檢表資料失敗: %v", err)
		} else {
			earningsResult = res
			data.EarningsSource = res.Source
			data.EarningsNote = res.Note
		}
	}

	grouped := revenue.GroupByYear(result.Records)
	years := make([]int, 0, len(grouped))
	for y := range grouped {
		years = append(years, y)
	}
	sort.Ints(years)
	data.AvailableYears = years

	activeYear := determineYear(query, years, a.now().Year())
	data.Year = activeYear

	var actualQuarters map[int]valuation.QuarterActual
	if len(earningsResult.Records) > 0 {
		actualQuarters = make(map[int]valuation.QuarterActual)
		for _, record := range earningsResult.Records {
			if record.Year == activeYear {
				actualQuarters[record.Quarter] = valuation.QuarterActual{
					NetIncome: record.NetIncome,
					EPS:       record.BasicEPS,
				}
			}
		}
		data.Earnings = buildEarningsView(earningsResult.Records, activeYear)
		if latest, label := latestEPSReference(earningsResult.Records); latest > 0 {
			data.EPSReference = fmt.Sprintf("%.2f", latest)
			data.EPSReferenceLabel = label
			if query.Get("prev_eps") == "" {
				form.PrevQuarterEPS = fmt.Sprintf("%.2f", latest)
				data.Form = form
			}
		}
	}

	manualYoY := parseYoYInputs(query)
	assumptions := form.toAssumptions()
	assumptions.ActualQuarters = actualQuarters
	projection, err := valuation.BuildYearProjection(activeYear, grouped, manualYoY, assumptions)
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
	StockNo           string
	Year              int
	AvailableYears    []int
	StockOptions      []stockOption
	Company           *companyView
	DataSource        string
	DataNote          string
	EarningsSource    string
	EarningsNote      string
	Form              formValues
	Projection        *valuation.YearProjection
	Months            []monthView
	Quarters          []quarterView
	Summary           summaryView
	Earnings          []earningsView
	EPSReference      string
	EPSReferenceLabel string
	Error             string
}

type stockOption struct {
	Code  string
	Label string
}

type companyView struct {
	StockNo     string
	Name        string
	Industry    string
	Website     string
	Description string
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

func buildForm(values url.Values, company *revenue.StaticCompany) formValues {
	return formValues{
		GrossMargin:        pickValue(values, "gross_margin", companyDefault(company, "gross_margin"), "17.3"),
		OperatingExpense:   pickValue(values, "operating_expense", companyDefault(company, "operating_expense"), "38"),
		NonOperatingIncome: pickValue(values, "non_operating_income", companyDefault(company, "non_operating_income"), "38"),
		TaxRate:            pickValue(values, "tax_rate", companyDefault(company, "tax_rate"), "20"),
		Shares:             pickValue(values, "shares", companyDefault(company, "shares"), "80"),
		PrevQuarterEPS:     pickValue(values, "prev_eps", companyDefault(company, "prev_eps"), "0.8"),
		PerMultiple:        pickValue(values, "per", companyDefault(company, "per"), "23"),
		CurrentPrice:       pickValue(values, "current_price", companyDefault(company, "current_price"), "56"),
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
	Index                int
	Label                string
	PreviousRevenue      float64
	PreviousMonthRevenue float64
	Revenue              float64
	YoYPercent           float64
	MoMPercent           float64
	IsActual             bool
	InputName            string
	InputValue           string
	Editable             bool
	InputID              string
	ReferenceYoYPercent  float64
	ReferenceMoMPercent  float64
	ReferenceRevenue     float64
	HasReference         bool
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

type earningsView struct {
	Year          int
	Quarter       int
	Label         string
	NetIncome     float64
	EPS           float64
	IsCurrentYear bool
}

type summaryView struct {
	AnnualRevenue  float64
	AnnualEPS      float64
	EstimatedPrice float64
	UpsidePercent  float64
	AvgYoYPercent  float64
	AvgMoMPercent  float64
}

func buildMonthViews(months []valuation.MonthEstimate, manual map[time.Month]float64) []monthView {
	views := make([]monthView, 0, len(months))
	for _, m := range months {
		name := fmt.Sprintf("yoy_%02d", int(m.Month))
		id := fmt.Sprintf("input-%02d", int(m.Month))
		inputValue := fmt.Sprintf("%.2f", m.YoY*100)
		if v, ok := manual[m.Month]; ok {
			inputValue = fmt.Sprintf("%.2f", v*100)
		}
		views = append(views, monthView{
			Index:                int(m.Month),
			Label:                fmt.Sprintf("%02d月", int(m.Month)),
			PreviousRevenue:      m.PreviousRevenue,
			PreviousMonthRevenue: m.PreviousMonthRevenue,
			Revenue:              m.Revenue,
			YoYPercent:           m.YoY * 100,
			MoMPercent:           m.MoM * 100,
			IsActual:             m.IsActual,
			InputName:            name,
			InputValue:           inputValue,
			Editable:             !m.IsActual,
			InputID:              id,
			ReferenceYoYPercent:  m.ReferenceYoY * 100,
			ReferenceMoMPercent:  m.ReferenceMoM * 100,
			ReferenceRevenue:     m.ReferenceRevenue,
			HasReference:         !m.IsActual && m.HasReference,
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
	avgMoMPercent := p.AvgMoM * 100
	return summaryView{
		AnnualRevenue:  p.AnnualRevenue,
		AnnualEPS:      p.AnnualEPS,
		EstimatedPrice: p.EstimatedPrice,
		UpsidePercent:  upsidePercent,
		AvgYoYPercent:  avgYoYPercent,
		AvgMoMPercent:  avgMoMPercent,
	}
}

func buildEarningsView(records []financials.QuarterlyReport, activeYear int) []earningsView {
	sorted := make([]financials.QuarterlyReport, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Year == sorted[j].Year {
			return sorted[i].Quarter < sorted[j].Quarter
		}
		return sorted[i].Year < sorted[j].Year
	})
	views := make([]earningsView, 0, len(sorted))
	for _, rec := range sorted {
		views = append(views, earningsView{
			Year:          rec.Year,
			Quarter:       rec.Quarter,
			Label:         fmt.Sprintf("%dQ%d", rec.Year, rec.Quarter),
			NetIncome:     rec.NetIncome,
			EPS:           rec.BasicEPS,
			IsCurrentYear: rec.Year == activeYear,
		})
	}
	return views
}

func latestEPSReference(records []financials.QuarterlyReport) (float64, string) {
	var (
		found  bool
		latest financials.QuarterlyReport
	)
	for _, rec := range records {
		if !found || rec.Year > latest.Year || (rec.Year == latest.Year && rec.Quarter > latest.Quarter) {
			latest = rec
			found = true
		}
	}
	if !found || latest.BasicEPS <= 0 {
		return 0, ""
	}
	return latest.BasicEPS, fmt.Sprintf("%dQ%d", latest.Year, latest.Quarter)
}

func buildStockOptions() []stockOption {
	companies := revenue.StaticCompanyList()
	options := make([]stockOption, 0, len(companies))
	for _, company := range companies {
		label := fmt.Sprintf("%s - %s", company.StockNo, company.Name)
		options = append(options, stockOption{Code: company.StockNo, Label: label})
	}
	return options
}

func toCompanyView(company *revenue.StaticCompany) *companyView {
	if company == nil {
		return nil
	}
	return &companyView{
		StockNo:     company.StockNo,
		Name:        company.Name,
		Industry:    company.Industry,
		Website:     company.Website,
		Description: company.Description,
	}
}

func determineYear(values url.Values, available []int, now int) int {
	if len(available) == 0 {
		if raw := values.Get("year"); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil {
				return v
			}
		}
		return now
	}
	if raw := values.Get("year"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			if containsInt(available, v) {
				return v
			}
		}
	}
	for i := len(available) - 1; i >= 0; i-- {
		if available[i] <= now {
			return available[i]
		}
	}
	return available[len(available)-1]
}

func containsInt(list []int, target int) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func companyDefault(company *revenue.StaticCompany, key string) string {
	if company == nil {
		return ""
	}
	if company.FormDefaults == nil {
		return ""
	}
	return company.FormDefaults[key]
}

func pickValue(values url.Values, key string, fallbacks ...string) string {
	if raw := values.Get(key); raw != "" {
		return raw
	}
	for _, fb := range fallbacks {
		if fb != "" {
			return fb
		}
	}
	return ""
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
