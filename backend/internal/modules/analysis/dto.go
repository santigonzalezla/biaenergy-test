package analysis

import (
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type StartRequest struct {
	MeterIds []string `json:"meterIds"`
}

type AnalysisResponse struct {
	Id             uuid.UUID         `json:"id"`
	NumId          int32             `json:"numId"`
	Status         db.AnalysisStatus `json:"status"`
	CurrentStep    *string           `json:"currentStep"`
	Progress       int16             `json:"progress"`
	MetersAnalyzed int32             `json:"metersAnalyzed"`
	Anomalies      int32             `json:"anomalies"`
	HighPriority   int32             `json:"highPriority"`
	Error          *string           `json:"error"`
	StartedAt      *time.Time        `json:"startedAt"`
	FinishedAt     *time.Time        `json:"finishedAt"`
	CreatedAt      time.Time         `json:"createdAt"`
}

func toAnalysisResponse(analysis db.Analysis) AnalysisResponse {
	return AnalysisResponse{
		Id:             analysis.UidAnalysis,
		NumId:          analysis.NumIDAnalysis,
		Status:         analysis.StrStatusAnalysis,
		CurrentStep:    analysis.StrCurrentStepAnalysis,
		Progress:       analysis.NumProgressAnalysis,
		MetersAnalyzed: analysis.NumMetersAnalyzed,
		Anomalies:      analysis.NumAnomaliesAnalysis,
		HighPriority:   analysis.NumHighPriorityAnalysis,
		Error:          analysis.StrErrorAnalysis,
		StartedAt:      utc(analysis.DtmStartedAtAnalysis),
		FinishedAt:     utc(analysis.DtmFinishedAtAnalysis),
		CreatedAt:      analysis.DtmCreatedAt.UTC(),
	}
}

func utc(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	converted := value.UTC()

	return &converted
}
