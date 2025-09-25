package revenue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const DefaultEndpoint = "https://openapi.twse.com.tw/v1/opendata/t187ap03_L"

var ErrNoData = errors.New("找不到符合的營收資料")

const (
	// SourceTWSE 表示資料來自台灣證券交易所開放資料平台。
	SourceTWSE = "TWSE 開放資料"
	// SourceFallback 表示資料改用內建示範資訊。
	SourceFallback = "內建示範資料"
)

// FetchResult 為營收資料取得的結果摘要。
type FetchResult struct {
	Records []MonthlyRevenue
	Source  string
	Company *StaticCompany
	Note    string
}


// Service 封裝對官方 API 的存取邏輯。
type Service struct {
	Client   *http.Client
	Endpoint string
}

// Fetch 取得指定股票代號的月營收。
func (s *Service) Fetch(ctx context.Context, stockNo string) (FetchResult, error) {
	key := strings.TrimSpace(stockNo)
	if key == "" {
		return FetchResult{}, fmt.Errorf("stockNo 為必填")
	}
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return FetchResult{}, fmt.Errorf("建立請求失敗: %w", err)
	}
	company, hasCompany := LookupStaticCompany(key)
	resp, err := client.Do(req)
	if err != nil {
		if hasCompany {
			note := fmt.Sprintf("API 連線失敗，改用內建示例資料: %v", err)
			return FetchResult{
				Records: SortMonthlyRevenues(cloneMonthlyRecords(company.Records)),
				Source:  SourceFallback,
				Company: company,
				Note:    note,
			}, nil
		}
		return FetchResult{}, fmt.Errorf("呼叫營收 API 失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if hasCompany {
			note := fmt.Sprintf("API 回傳狀態碼 %d，改用內建示例資料", resp.StatusCode)
			if len(body) > 0 {
				note = fmt.Sprintf("%s: %s", note, string(body))
			}
			return FetchResult{
				Records: SortMonthlyRevenues(cloneMonthlyRecords(company.Records)),
				Source:  SourceFallback,
				Company: company,
				Note:    note,
			}, nil
		}
		return FetchResult{}, fmt.Errorf("營收 API 回傳狀態碼 %d: %s", resp.StatusCode, string(body))
	}
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchResult{}, fmt.Errorf("讀取 API 回傳內容失敗: %w", err)
	}
	var rawRecords []RawRecord
	if err := json.Unmarshal(rawBody, &rawRecords); err != nil {
		return FetchResult{}, fmt.Errorf("解析營收 JSON 失敗: %w", err)
	}
	filtered := FilterByStock(rawRecords, key)
	if len(filtered) == 0 {
		if hasCompany {
			return FetchResult{
				Records: SortMonthlyRevenues(cloneMonthlyRecords(company.Records)),
				Source:  SourceFallback,
				Company: company,
				Note:    "官方資料暫無該公司紀錄，改用內建示例資料",
			}, nil
		}
		return FetchResult{}, ErrNoData

	}
	normalized := make([]MonthlyRevenue, 0, len(filtered))
	for _, rec := range filtered {
		value, err := rec.Normalize()
		if err != nil {
			return FetchResult{}, err
		}
		normalized = append(normalized, value)
	}
	note := "資料來自台灣證券交易所開放資料"
	if hasCompany {
		note += "；表單預設值會自動載入該公司常見範例"
	}
	return FetchResult{
		Records: SortMonthlyRevenues(normalized),
		Source:  SourceTWSE,
		Company: company,
		Note:    note,
	}, nil

}
