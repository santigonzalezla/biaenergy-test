package dashboard

import (
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type SummaryResponse struct {
	MetersCount           int64                 `json:"metersCount"`
	MetersWithAlerts      int64                 `json:"metersWithAlerts"`
	ReadingsCount         int64                 `json:"readingsCount"`
	TotalKwh              float64               `json:"totalKwh"`
	OpenAnomalies         int64                 `json:"openAnomalies"`
	HighPriorityAnomalies int64                 `json:"highPriorityAnomalies"`
	LastAnalysis          *LastAnalysisResponse `json:"lastAnalysis"`
}

type LastAnalysisResponse struct {
	Id           uuid.UUID         `json:"id"`
	Status       db.AnalysisStatus `json:"status"`
	CurrentStep  *string           `json:"currentStep"`
	Progress     int16             `json:"progress"`
	Anomalies    int32             `json:"anomalies"`
	HighPriority int32             `json:"highPriority"`
	Error        *string           `json:"error"`
	CreatedAt    time.Time         `json:"createdAt"`
	FinishedAt   *time.Time        `json:"finishedAt"`
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

func toLastAnalysisResponse(analysis db.Analysis) *LastAnalysisResponse {
	var finishedAt *time.Time

	if analysis.DtmFinishedAtAnalysis != nil {
		utc := analysis.DtmFinishedAtAnalysis.UTC()
		finishedAt = &utc
	}

	return &LastAnalysisResponse{
		Id:           analysis.UidAnalysis,
		Status:       analysis.StrStatusAnalysis,
		CurrentStep:  analysis.StrCurrentStepAnalysis,
		Progress:     analysis.NumProgressAnalysis,
		Anomalies:    analysis.NumAnomaliesAnalysis,
		HighPriority: analysis.NumHighPriorityAnalysis,
		Error:        analysis.StrErrorAnalysis,
		CreatedAt:    analysis.DtmCreatedAt.UTC(),
		FinishedAt:   finishedAt,
	}
}
