package anomaly

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type ListQuery struct {
	AnalysisId string
	Type       string
	Severity   string
	Status     string
}

type UpdateStatusRequest struct {
	Status string  `json:"status"`
	Note   *string `json:"note"`
}

type ListResponse struct {
	AnalysisId *uuid.UUID        `json:"analysisId"`
	Data       []SummaryResponse `json:"data"`
}

type SummaryResponse struct {
	Id            uuid.UUID          `json:"id"`
	NumId         int32              `json:"numId"`
	MeterId       uuid.UUID          `json:"meterId"`
	MeterCode     string             `json:"meterCode"`
	MeterName     string             `json:"meterName"`
	Type          db.AnomalyType     `json:"type"`
	Severity      db.AnomalySeverity `json:"severity"`
	Status        db.AnomalyStatus   `json:"status"`
	RuleId        string             `json:"ruleId"`
	Confidence    float64            `json:"confidence"`
	PriorityScore float64            `json:"priorityScore"`
	VariationPct  *float64           `json:"variationPct"`
	DetectedAt    time.Time          `json:"detectedAt"`
	Reason        string             `json:"reason"`
}

type MeterReference struct {
	Id       uuid.UUID `json:"id"`
	Code     string    `json:"code"`
	Name     string    `json:"name"`
	Location string    `json:"location"`
}

type EventReference struct {
	Id          uuid.UUID    `json:"id"`
	Type        db.EventType `json:"type"`
	Timestamp   time.Time    `json:"timestamp"`
	Description string       `json:"description"`
}

type DetailResponse struct {
	Id                uuid.UUID          `json:"id"`
	NumId             int32              `json:"numId"`
	AnalysisId        uuid.UUID          `json:"analysisId"`
	Meter             MeterReference     `json:"meter"`
	Type              db.AnomalyType     `json:"type"`
	Severity          db.AnomalySeverity `json:"severity"`
	Status            db.AnomalyStatus   `json:"status"`
	RuleId            string             `json:"ruleId"`
	Confidence        float64            `json:"confidence"`
	PriorityScore     float64            `json:"priorityScore"`
	BaselineKwh       *float64           `json:"baselineKwh"`
	CurrentKwh        *float64           `json:"currentKwh"`
	VariationPct      *float64           `json:"variationPct"`
	DetectedAt        time.Time          `json:"detectedAt"`
	WindowStart       *time.Time         `json:"windowStart"`
	WindowEnd         *time.Time         `json:"windowEnd"`
	Reason            string             `json:"reason"`
	RecommendedAction string             `json:"recommendedAction"`
	ChangedVariables  []string           `json:"changedVariables"`
	Evidence          json.RawMessage    `json:"evidence"`
	RelatedEvent      *EventReference    `json:"relatedEvent"`
	ResolutionNote    *string            `json:"resolutionNote"`
	UpdatedAt         time.Time          `json:"updatedAt"`
}

func toSummaryResponses(rows []db.ListAnomaliesRow) []SummaryResponse {
	summaries := make([]SummaryResponse, 0, len(rows))

	for _, row := range rows {
		summaries = append(summaries, SummaryResponse{
			Id:            row.UidAnomaly,
			NumId:         row.NumIDAnomaly,
			MeterId:       row.UidMeter,
			MeterCode:     row.StrCodeMeter,
			MeterName:     row.StrNameMeter,
			Type:          row.StrTypeAnomaly,
			Severity:      row.StrSeverityAnomaly,
			Status:        row.StrStatusAnomaly,
			RuleId:        row.StrRuleIDAnomaly,
			Confidence:    row.DecConfidenceAnomaly,
			PriorityScore: row.DecPriorityScoreAnomaly,
			VariationPct:  row.DecVariationPctAnomaly,
			DetectedAt:    row.DtmDetectedAtAnomaly.UTC(),
			Reason:        row.StrReasonAnomaly,
		})
	}

	return summaries
}

func toDetailResponse(row db.GetAnomalyRow) DetailResponse {
	return DetailResponse{
		Id:         row.UidAnomaly,
		NumId:      row.NumIDAnomaly,
		AnalysisId: row.UidAnalysis,
		Meter: MeterReference{
			Id:       row.UidMeter,
			Code:     row.StrCodeMeter,
			Name:     row.StrNameMeter,
			Location: row.StrLocationMeter,
		},
		Type:              row.StrTypeAnomaly,
		Severity:          row.StrSeverityAnomaly,
		Status:            row.StrStatusAnomaly,
		RuleId:            row.StrRuleIDAnomaly,
		Confidence:        row.DecConfidenceAnomaly,
		PriorityScore:     row.DecPriorityScoreAnomaly,
		BaselineKwh:       row.DecBaselineKwhAnomaly,
		CurrentKwh:        row.DecCurrentKwhAnomaly,
		VariationPct:      row.DecVariationPctAnomaly,
		DetectedAt:        row.DtmDetectedAtAnomaly.UTC(),
		WindowStart:       utc(row.DtmWindowStartAnomaly),
		WindowEnd:         utc(row.DtmWindowEndAnomaly),
		Reason:            row.StrReasonAnomaly,
		RecommendedAction: row.StrRecommendedActionAnomaly,
		ChangedVariables:  row.ArrChangedVariablesAnomaly,
		Evidence:          row.JsonEvidenceAnomaly,
		RelatedEvent:      toEventReference(row),
		ResolutionNote:    row.StrResolutionNoteAnomaly,
		UpdatedAt:         row.DtmUpdatedAt.UTC(),
	}
}

func toEventReference(row db.GetAnomalyRow) *EventReference {
	if row.UidEvent == nil || row.StrTypeEvent == nil || row.DtmTimestampEvent == nil {
		return nil
	}

	description := ""
	if row.StrDescriptionEvent != nil {
		description = *row.StrDescriptionEvent
	}

	return &EventReference{
		Id:          *row.UidEvent,
		Type:        *row.StrTypeEvent,
		Timestamp:   row.DtmTimestampEvent.UTC(),
		Description: description,
	}
}

func utc(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	converted := value.UTC()

	return &converted
}
