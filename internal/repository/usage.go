package repository

import (
	"context"
	"math"

	"taskmanager/models"
)

// ModelUsage aggregates attempts per provider/model and computes each model's
// share of the total tokens.
func (s *Store) ModelUsage(ctx context.Context) (models.ModelUsageResponse, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider, model, COUNT(*),
		COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(cache_read_tokens), 0), COALESCE(SUM(cache_write_tokens), 0),
		COALESCE(SUM(cost_usd), 0), COALESCE(AVG(latency_ms), 0)
		FROM attempts WHERE status IN ('ok', 'error') GROUP BY provider, model
		ORDER BY SUM(input_tokens + output_tokens) DESC, provider, model`)
	if err != nil {
		return models.ModelUsageResponse{}, err
	}
	defer rows.Close()

	out := models.ModelUsageResponse{Items: []models.ModelUsage{}}
	for rows.Next() {
		var item models.ModelUsage
		var avg float64
		if err := rows.Scan(&item.Provider, &item.Model, &item.Calls, &item.InputTokens,
			&item.OutputTokens, &item.CacheReadTokens, &item.CacheWriteTokens, &item.CostUSD, &avg); err != nil {
			return models.ModelUsageResponse{}, err
		}
		item.Tokens = item.InputTokens + item.OutputTokens
		item.AvgLatencyMS = int64(math.Round(avg))
		out.TotalTokens += item.Tokens
		out.TotalCost += item.CostUSD
		out.Items = append(out.Items, item)
	}
	if err := rows.Err(); err != nil {
		return models.ModelUsageResponse{}, err
	}
	if out.TotalTokens > 0 {
		for i := range out.Items {
			out.Items[i].Share = math.Round(float64(out.Items[i].Tokens)/float64(out.TotalTokens)*1000) / 10
		}
	}
	return out, nil
}
