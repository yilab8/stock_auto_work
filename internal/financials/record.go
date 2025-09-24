package financials

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// RawQuarterRecord 以字典形式保留檢表原始欄位。
type RawQuarterRecord map[string]string

// Value 依序取得第一個非空欄位值。
func (r RawQuarterRecord) Value(keys ...string) string {
	for _, key := range keys {
		if v, ok := r[key]; ok {
			trimmed := strings.TrimSpace(v)
			if trimmed != "" && trimmed != "-" {
				return trimmed
			}
		}
	}
	return ""
}

// QuarterlyReport 代表單季稅後淨利與 EPS 資料。
type QuarterlyReport struct {
	CompanyCode string
	Year        int
	Quarter     int
	NetIncome   float64
	BasicEPS    float64
}

// Normalize 轉換原始欄位為結構化資料。
func (r RawQuarterRecord) Normalize() (QuarterlyReport, error) {
	yearRaw := r.Value("年度", "年")
	if yearRaw == "" {
		return QuarterlyReport{}, fmt.Errorf("缺少年度欄位")
	}
	year, err := strconv.Atoi(yearRaw)
	if err != nil {
		return QuarterlyReport{}, fmt.Errorf("年度格式錯誤: %w", err)
	}
	quarterRaw := r.Value("季別", "季", "季度")
	if quarterRaw == "" {
		return QuarterlyReport{}, fmt.Errorf("缺少季別欄位")
	}
	quarter, err := parseQuarter(quarterRaw)
	if err != nil {
		return QuarterlyReport{}, err
	}
	netIncomeRaw := r.Value("稅後淨利", "綜合損益總額-稅後淨利", "本期稅後淨利", "本期綜合損益總額", "歸屬於母公司業主之淨利(損失)")
	if netIncomeRaw == "" {
		return QuarterlyReport{}, fmt.Errorf("缺少稅後淨利欄位")
	}
	netIncome, err := parseNumber(netIncomeRaw)
	if err != nil {
		return QuarterlyReport{}, fmt.Errorf("稅後淨利格式錯誤: %w", err)
	}
	epsRaw := r.Value("基本每股盈餘", "基本每股盈餘(元)", "基本每股盈餘-完全稀釋")
	eps := 0.0
	if epsRaw != "" {
		eps, err = parseNumber(epsRaw)
		if err != nil {
			return QuarterlyReport{}, fmt.Errorf("EPS 格式錯誤: %w", err)
		}
	}
	return QuarterlyReport{
		CompanyCode: r.Value("公司代號"),
		Year:        year,
		Quarter:     quarter,
		NetIncome:   netIncome,
		BasicEPS:    eps,
	}, nil
}

func parseQuarter(raw string) (int, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "第")
	cleaned = strings.TrimSuffix(cleaned, "季")
	cleaned = strings.TrimSuffix(cleaned, "Q")
	cleaned = strings.TrimPrefix(cleaned, "Q")
	cleaned = strings.TrimSpace(cleaned)
	v, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0, fmt.Errorf("季別格式錯誤: %w", err)
	}
	if v < 1 || v > 4 {
		return 0, fmt.Errorf("季別超出範圍: %d", v)
	}
	return v, nil
}

func parseNumber(raw string) (float64, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "元", "")
	cleaned = strings.ReplaceAll(cleaned, "千元", "")
	if cleaned == "" {
		return 0, fmt.Errorf("空值")
	}
	return strconv.ParseFloat(cleaned, 64)
}

// SortQuarterlyReports 依年份與季度排序。
func SortQuarterlyReports(records []QuarterlyReport) []QuarterlyReport {
	sorted := make([]QuarterlyReport, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Year == sorted[j].Year {
			return sorted[i].Quarter < sorted[j].Quarter
		}
		return sorted[i].Year < sorted[j].Year
	})
	return sorted
}

// FilterByStock 篩選指定公司代號的紀錄。
func FilterByStock(records []RawQuarterRecord, stockNo string) []RawQuarterRecord {
	key := strings.TrimSpace(stockNo)
	if key == "" {
		return nil
	}
	var out []RawQuarterRecord
	for _, rec := range records {
		if strings.EqualFold(strings.TrimSpace(rec.Value("公司代號")), key) {
			out = append(out, rec)
		}
	}
	return out
}
