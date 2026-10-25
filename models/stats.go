package models

import "time"

// Stats is the dashboard summary.
type Stats struct {
	Total      int     `json:"total"`
	Running    int     `json:"running"`
	Done       int     `json:"done"`
	Failed     int     `json:"failed"`
	NeedsHuman int     `json:"needs_human"`
	CostUSD    float64 `json:"cost_usd"`
	Tokens     int     `json:"tokens"`
}

// ModelUsage is token accounting for one provider/model pair.
type ModelUsage struct {
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	Calls            int     `json:"calls"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	CacheReadTokens  int     `json:"cache_read_tokens"`
	CacheWriteTokens int     `json:"cache_write_tokens"`
	Tokens           int     `json:"tokens"`
	CostUSD          float64 `json:"cost_usd"`
	AvgLatencyMS     int64   `json:"avg_latency_ms"`
	Share            float64 `json:"share"`
	TodayTokens      int     `json:"today_tokens"`
	TodayCalls       int     `json:"today_calls"`
}

// ModelHealth reports the latest provider refusal for one provider/model pair.
type ModelHealth struct {
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	Status      string    `json:"status"`
	LastError   string    `json:"last_error"`
	LastErrorAt time.Time `json:"last_error_at"`
}

// ModelUsageResponse is the per-model usage read model.
type ModelUsageResponse struct {
	Items       []ModelUsage `json:"items"`
	TotalTokens int          `json:"total_tokens"`
	TotalCost   float64      `json:"total_cost"`
	DailyTokens int          `json:"daily_tokens"`
	DailyCost   float64      `json:"daily_cost"`
}
