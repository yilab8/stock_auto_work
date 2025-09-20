package revenue

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RawRecord 代表官方開放資料中的原始欄位。
type RawRecord struct {
	CompanyCode    string `json:"公司代號"`
	CompanyName    string `json:"公司名稱"`
	PublishDate    string `json:"出表日期"`
	DataMonth      string `json:"資料年月"`
	MonthlyRevenue string `json:"營業收入-當月營收"`
	AccRevenue     string `json:"營業收入-當月累計營收"`
	AccRevenueLast string `json:"營業收入-去年累計營收"`
	YoY            string `json:"營業收入-去年同月增減(%)"`
	Note           string `json:"備註"`
}

// MonthlyRevenue 表示整理後的單月營收資訊。
type MonthlyRevenue struct {
	Year    int
	Month   time.Month
	Revenue float64
}

// ParseYearMonth 解析民國或西元格式的年月。
func (r RawRecord) ParseYearMonth() (int, time.Month, error) {
	raw := strings.TrimSpace(r.DataMonth)
	if raw == "" {
		return 0, 0, fmt.Errorf("資料年月為空值")
	}
	// 部分資料使用民國年 (例如 11201)，也可能使用西元 (例如 202401)
	if len(raw) != 5 && len(raw) != 6 {
		return 0, 0, fmt.Errorf("未知的年月格式: %s", raw)
	}
	yearPart := raw[:len(raw)-2]
	monthPart := raw[len(raw)-2:]

	monthNum, err := strconv.Atoi(monthPart)
	if err != nil {
		return 0, 0, fmt.Errorf("月份格式錯誤: %w", err)
	}
	if monthNum < 1 || monthNum > 12 {
		return 0, 0, fmt.Errorf("月份不在 1-12 範圍內: %d", monthNum)
	}
	yearNum, err := strconv.Atoi(yearPart)
	if err != nil {
		return 0, 0, fmt.Errorf("年份格式錯誤: %w", err)
	}
	if len(raw) == 5 { // 民國年
		yearNum += 1911
	}
	return yearNum, time.Month(monthNum), nil
}

// ParseRevenue 解析字串金額並轉換為 float64 (單位: 新台幣千元)。
func (r RawRecord) ParseRevenue() (float64, error) {
	raw := strings.TrimSpace(r.MonthlyRevenue)
	if raw == "" || raw == "-" {
		return 0, fmt.Errorf("缺少當月營收資料")
	}
	raw = strings.ReplaceAll(raw, ",", "")
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("營收資料格式錯誤: %w", err)
	}
	return value, nil
}

// Normalize 將 RawRecord 轉為 MonthlyRevenue。
func (r RawRecord) Normalize() (MonthlyRevenue, error) {
	year, month, err := r.ParseYearMonth()
	if err != nil {
		return MonthlyRevenue{}, err
	}
	value, err := r.ParseRevenue()
	if err != nil {
		return MonthlyRevenue{}, err
	}
	return MonthlyRevenue{
		Year:    year,
		Month:   month,
		Revenue: value,
	}, nil
}

// FilterByStock 依照公司代號過濾資料。
func FilterByStock(records []RawRecord, stockNo string) []RawRecord {
	stockNo = strings.TrimSpace(stockNo)
	var out []RawRecord
	for _, rec := range records {
		if strings.EqualFold(strings.TrimSpace(rec.CompanyCode), stockNo) {
			out = append(out, rec)
		}
	}
	return out
}

// GroupByYear 將資料依年份分組。
func GroupByYear(records []MonthlyRevenue) map[int][]MonthlyRevenue {
	result := make(map[int][]MonthlyRevenue)
	for _, rec := range records {
		yearList := result[rec.Year]
		yearList = append(yearList, rec)
		result[rec.Year] = yearList
	}
	return result
}

// SortMonthlyRevenues 依月份排序資料。
func SortMonthlyRevenues(records []MonthlyRevenue) []MonthlyRevenue {
	sorted := make([]MonthlyRevenue, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Year == sorted[j].Year {
			return sorted[i].Month < sorted[j].Month
		}
		return sorted[i].Year < sorted[j].Year
	})
	return sorted
}
