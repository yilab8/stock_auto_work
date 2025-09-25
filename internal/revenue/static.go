package revenue

import (
	"sort"
	"strings"
	"time"
)

// StaticCompany 提供內建示範用的上市櫃公司資料。
type StaticCompany struct {
	StockNo      string
	Name         string
	Industry     string
	Website      string
	Description  string
	FormDefaults map[string]string
	Records      []MonthlyRevenue
}

var staticCompanies = map[string]*StaticCompany{
	"2330": {
		StockNo:  "2330",
		Name:     "台灣積體電路製造股份有限公司",
		Industry: "半導體",
		Website:  "https://www.tsmc.com/",
		Description: "台積電為全球晶圓代工龍頭，主要提供先進製程委外製造服務。" +
			"以下內建資料整理自 2023-2024 年公開月營收公告 (單位：新台幣千元)。",
		FormDefaults: map[string]string{
			"gross_margin":         "53.0",
			"operating_expense":    "40000000",
			"non_operating_income": "5000000",
			"tax_rate":             "14.0",
			"shares":               "25930",
			"prev_eps":             "7.51",
			"per":                  "24",
			"current_price":        "610",
		},
		Records: []MonthlyRevenue{
			newMonthlyRevenue(2023, time.January, 200070000),
			newMonthlyRevenue(2023, time.February, 163174000),
			newMonthlyRevenue(2023, time.March, 145407000),
			newMonthlyRevenue(2023, time.April, 147904000),
			newMonthlyRevenue(2023, time.May, 176536000),
			newMonthlyRevenue(2023, time.June, 156404000),
			newMonthlyRevenue(2023, time.July, 186040000),
			newMonthlyRevenue(2023, time.August, 188693000),
			newMonthlyRevenue(2023, time.September, 211490000),
			newMonthlyRevenue(2023, time.October, 243208000),
			newMonthlyRevenue(2023, time.November, 226021000),
			newMonthlyRevenue(2023, time.December, 241681000),
			newMonthlyRevenue(2024, time.January, 215804000),
			newMonthlyRevenue(2024, time.February, 181648000),
			newMonthlyRevenue(2024, time.March, 195207000),
			newMonthlyRevenue(2024, time.April, 236021000),
			newMonthlyRevenue(2024, time.May, 247194000),
			newMonthlyRevenue(2024, time.June, 207878000),
		},
	},
	"2317": {
		StockNo:     "2317",
		Name:        "鴻海精密工業股份有限公司",
		Industry:    "電子 - 代工製造",
		Website:     "https://www.foxconntech.com.tw/",
		Description: "鴻海為全球最大電子代工與製造服務供應商，內建數據整理自 2023-2024 年公布月營收 (單位：新台幣千元)。",
		FormDefaults: map[string]string{
			"gross_margin":         "6.5",
			"operating_expense":    "26000000",
			"non_operating_income": "8000000",
			"tax_rate":             "18.0",
			"shares":               "13880",
			"prev_eps":             "1.40",
			"per":                  "12",
			"current_price":        "105",
		},
		Records: []MonthlyRevenue{
			newMonthlyRevenue(2023, time.January, 459132000),
			newMonthlyRevenue(2023, time.February, 402014000),
			newMonthlyRevenue(2023, time.March, 400349000),
			newMonthlyRevenue(2023, time.April, 450706000),
			newMonthlyRevenue(2023, time.May, 486675000),
			newMonthlyRevenue(2023, time.June, 490688000),
			newMonthlyRevenue(2023, time.July, 469234000),
			newMonthlyRevenue(2023, time.August, 441703000),
			newMonthlyRevenue(2023, time.September, 659209000),
			newMonthlyRevenue(2023, time.October, 741028000),
			newMonthlyRevenue(2023, time.November, 650088000),
			newMonthlyRevenue(2023, time.December, 629383000),
			newMonthlyRevenue(2024, time.January, 522093000),
			newMonthlyRevenue(2024, time.February, 447024000),
			newMonthlyRevenue(2024, time.March, 474746000),
			newMonthlyRevenue(2024, time.April, 510215000),
			newMonthlyRevenue(2024, time.May, 450035000),
			newMonthlyRevenue(2024, time.June, 568080000),
		},
	},
}

func newMonthlyRevenue(year int, month time.Month, value float64) MonthlyRevenue {
	return MonthlyRevenue{Year: year, Month: month, Revenue: value}
}

// LookupStaticCompany 依股票代號取得示例資料。
func LookupStaticCompany(stockNo string) (*StaticCompany, bool) {
	key := strings.TrimSpace(stockNo)
	if key == "" {
		return nil, false
	}
	if company, ok := staticCompanies[key]; ok {
		return company, true
	}
	return nil, false
}

// StaticCompanyList 取得內建示例公司的清單 (依股票代號排序)。
func StaticCompanyList() []*StaticCompany {
	out := make([]*StaticCompany, 0, len(staticCompanies))
	for _, company := range staticCompanies {
		out = append(out, company)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StockNo < out[j].StockNo })
	return out
}

// cloneMonthlyRecords 避免直接修改內建資料。
func cloneMonthlyRecords(records []MonthlyRevenue) []MonthlyRevenue {
	cloned := make([]MonthlyRevenue, len(records))
	copy(cloned, records)
	return cloned
}
