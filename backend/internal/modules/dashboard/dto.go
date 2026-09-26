package dashboard

import "github.com/santigonzalezla/biaenergy-test/backend/internal/db"

type SummaryResponse struct {
	MetersCount           int64   `json:"metersCount"`
	MetersWithAlerts      int64   `json:"metersWithAlerts"`
	ReadingsCount         int64   `json:"readingsCount"`
	TotalKwh              float64 `json:"totalKwh"`
	OpenAnomalies         int64   `json:"openAnomalies"`
	HighPriorityAnomalies int64   `json:"highPriorityAnomalies"`
}

func toSummaryResponse(summary db.GetDashboardSummaryRow) SummaryResponse {
	return SummaryResponse{
		MetersCount:           summary.MetersCount,
		MetersWithAlerts:      summary.MetersWithAlerts,
		ReadingsCount:         summary.ReadingsCount,
		TotalKwh:              summary.TotalKwh,
		OpenAnomalies:         summary.OpenAnomalies,
		HighPriorityAnomalies: summary.HighPriorityAnomalies,
	}
}
