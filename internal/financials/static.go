package financials

import "strings"

const (
	// SourceTWSE 代表資料來自官方 API。
	SourceTWSE = "TWSE 檢表開放資料"
	// SourceFallback 表示改用內建示例資料。
	SourceFallback = "內建檢表示例"
)

// StaticEarnings 提供示例稅後淨利資料。
type StaticEarnings struct {
	Records []QuarterlyReport
}

var staticEarnings = map[string]*StaticEarnings{
	"2330": {
		Records: []QuarterlyReport{
			{CompanyCode: "2330", Year: 2023, Quarter: 1, NetIncome: 206991000, BasicEPS: 7.98},
			{CompanyCode: "2330", Year: 2023, Quarter: 2, NetIncome: 181802000, BasicEPS: 7.01},
			{CompanyCode: "2330", Year: 2023, Quarter: 3, NetIncome: 211089000, BasicEPS: 8.14},
			{CompanyCode: "2330", Year: 2023, Quarter: 4, NetIncome: 238706000, BasicEPS: 8.14},
			{CompanyCode: "2330", Year: 2024, Quarter: 1, NetIncome: 225514000, BasicEPS: 8.70},
			{CompanyCode: "2330", Year: 2024, Quarter: 2, NetIncome: 236327000, BasicEPS: 9.00},
		},
	},
	"2317": {
		Records: []QuarterlyReport{
			{CompanyCode: "2317", Year: 2023, Quarter: 1, NetIncome: 20021400, BasicEPS: 1.46},
			{CompanyCode: "2317", Year: 2023, Quarter: 2, NetIncome: 33214300, BasicEPS: 2.28},
			{CompanyCode: "2317", Year: 2023, Quarter: 3, NetIncome: 48942500, BasicEPS: 3.36},
			{CompanyCode: "2317", Year: 2023, Quarter: 4, NetIncome: 42164600, BasicEPS: 2.90},
			{CompanyCode: "2317", Year: 2024, Quarter: 1, NetIncome: 22811900, BasicEPS: 1.57},
			{CompanyCode: "2317", Year: 2024, Quarter: 2, NetIncome: 28073500, BasicEPS: 1.93},
		},
	},
}

// LookupStaticEarnings 取得示例資料。
func LookupStaticEarnings(stockNo string) (*StaticEarnings, bool) {
	key := strings.TrimSpace(stockNo)
	if key == "" {
		return nil, false
	}
	item, ok := staticEarnings[key]
	return item, ok
}

func cloneQuarterlyReports(records []QuarterlyReport) []QuarterlyReport {
	cloned := make([]QuarterlyReport, len(records))
	copy(cloned, records)
	return cloned
}
