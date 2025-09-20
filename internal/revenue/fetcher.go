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

// Service 封裝對官方 API 的存取邏輯。
type Service struct {
	Client   *http.Client
	Endpoint string
}

// Fetch 取得指定股票代號的月營收。
func (s *Service) Fetch(ctx context.Context, stockNo string) ([]MonthlyRevenue, error) {
	if strings.TrimSpace(stockNo) == "" {
		return nil, fmt.Errorf("stockNo 為必填")
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
		return nil, fmt.Errorf("建立請求失敗: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("呼叫營收 API 失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("營收 API 回傳狀態碼 %d: %s", resp.StatusCode, string(body))
	}
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("讀取 API 回傳內容失敗: %w", err)
	}
	var rawRecords []RawRecord
	if err := json.Unmarshal(rawBody, &rawRecords); err != nil {
		return nil, fmt.Errorf("解析營收 JSON 失敗: %w", err)
	}
	filtered := FilterByStock(rawRecords, stockNo)
	if len(filtered) == 0 {
		return nil, ErrNoData
	}
	normalized := make([]MonthlyRevenue, 0, len(filtered))
	for _, rec := range filtered {
		value, err := rec.Normalize()
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, value)
	}
	return SortMonthlyRevenues(normalized), nil
}
