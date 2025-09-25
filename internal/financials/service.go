package financials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const DefaultEndpoint = "https://openapi.twse.com.tw/v1/opendata/t187ap08_E"

var ErrNoData = errors.New("找不到符合的檢表資料")

// FetchResult 彙整稅後淨利抓取結果。
type FetchResult struct {
	Records []QuarterlyReport
	Source  string
	Note    string
}

// Service 代表檢表資料抓取器。
type Service struct {
	Client   *http.Client
	Endpoint string
}

// Fetch 依股票代號取得稅後淨利資料。
func (s *Service) Fetch(ctx context.Context, stockNo string) (FetchResult, error) {
	key := strings.TrimSpace(stockNo)
	if key == "" {
		return FetchResult{}, fmt.Errorf("stockNo 為必填")
	}
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return FetchResult{}, fmt.Errorf("建立請求失敗: %w", err)
	}
	static, hasStatic := LookupStaticEarnings(key)
	resp, err := client.Do(req)
	if err != nil {
		if hasStatic {
			return FetchResult{
				Records: SortQuarterlyReports(cloneQuarterlyReports(static.Records)),
				Source:  SourceFallback,
				Note:    fmt.Sprintf("API 連線失敗，改用內建檢表: %v", err),
			}, nil
		}
		return FetchResult{}, fmt.Errorf("呼叫檢表 API 失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if hasStatic {
			note := fmt.Sprintf("API 回傳狀態碼 %d，改用內建檢表", resp.StatusCode)
			if len(body) > 0 {
				note = fmt.Sprintf("%s: %s", note, string(body))
			}
			return FetchResult{
				Records: SortQuarterlyReports(cloneQuarterlyReports(static.Records)),
				Source:  SourceFallback,
				Note:    note,
			}, nil
		}
		return FetchResult{}, fmt.Errorf("檢表 API 回傳狀態碼 %d: %s", resp.StatusCode, string(body))
	}
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchResult{}, fmt.Errorf("讀取檢表 API 回傳失敗: %w", err)
	}
	var rawRecords []RawQuarterRecord
	if err := json.Unmarshal(rawBody, &rawRecords); err != nil {
		return FetchResult{}, fmt.Errorf("解析檢表 JSON 失敗: %w", err)
	}
	filtered := FilterByStock(rawRecords, key)
	if len(filtered) == 0 {
		if hasStatic {
			return FetchResult{
				Records: SortQuarterlyReports(cloneQuarterlyReports(static.Records)),
				Source:  SourceFallback,
				Note:    "官方資料暫無該公司檢表，改用內建示例",
			}, nil
		}
		return FetchResult{}, ErrNoData
	}
	normalized := make([]QuarterlyReport, 0, len(filtered))
	for _, rec := range filtered {
		value, err := rec.Normalize()
		if err != nil {
			return FetchResult{}, err
		}
		normalized = append(normalized, value)
	}
	return FetchResult{
		Records: SortQuarterlyReports(normalized),
		Source:  SourceTWSE,
		Note:    "資料來自台灣證券交易所檢表開放資料",
	}, nil
}
