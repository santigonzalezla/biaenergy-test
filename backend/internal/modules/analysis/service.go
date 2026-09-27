package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/aiclient"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

const (
	stepLoading   = "loading_data"
	stepAnalyzing = "analyzing"
	stepSaving    = "saving"

	progressLoading   = 10
	progressAnalyzing = 40
	progressSaving    = 85

	failureWriteTimeout = 5 * time.Second
)

type Analyzer interface {
	Analyze(ctx context.Context, request aiclient.AnalyzeRequest) (aiclient.AnalyzeResult, error)
}

type Service struct {
	lifetime   context.Context
	repository Repository
	analyzer   Analyzer
	timezone   string
	timeout    time.Duration
	startMutex sync.Mutex
	workers    sync.WaitGroup
}

func NewService(lifetime context.Context, repository Repository, analyzer Analyzer, timezone string, timeout time.Duration) *Service {
	return &Service{
		lifetime:   lifetime,
		repository: repository,
		analyzer:   analyzer,
		timezone:   timezone,
		timeout:    timeout,
	}
}

func (service *Service) Start(ctx context.Context, request StartRequest) (AnalysisResponse, bool, error) {
	meterIds, err := parseMeterIds(request.MeterIds)

	if err != nil {
		return AnalysisResponse{}, false, err
	}

	service.startMutex.Lock()
	defer service.startMutex.Unlock()

	active, running, err := service.repository.Active(ctx)

	if err != nil {
		return AnalysisResponse{}, false, err
	}

	if running {
		return toAnalysisResponse(active), false, nil
	}

	created, err := service.repository.Create(ctx)

	if err != nil {
		return AnalysisResponse{}, false, err
	}

	service.workers.Add(1)
	go service.run(created.UidAnalysis, meterIds)

	return toAnalysisResponse(created), true, nil
}

func (service *Service) Get(ctx context.Context, id uuid.UUID) (AnalysisResponse, error) {
	analysis, err := service.repository.GetById(ctx, id)

	if errors.Is(err, ErrNotFound) {
		return AnalysisResponse{}, apperror.NotFound("ANALYSIS_NOT_FOUND", "Analysis not found")
	}

	if err != nil {
		return AnalysisResponse{}, err
	}

	return toAnalysisResponse(analysis), nil
}

func (service *Service) Latest(ctx context.Context) (AnalysisResponse, error) {
	analysis, exists, err := service.repository.Latest(ctx)

	if err != nil {
		return AnalysisResponse{}, err
	}

	if !exists {
		return AnalysisResponse{}, apperror.NotFound("NO_ANALYSIS_YET", "No analysis has been run yet")
	}

	return toAnalysisResponse(analysis), nil
}

func (service *Service) RecoverInterrupted(ctx context.Context) error {
	interrupted, err := service.repository.FailInterrupted(ctx)

	if err != nil {
		return err
	}

	if interrupted > 0 {
		slog.Warn("closed analyses interrupted by a restart", "count", interrupted)
	}

	return nil
}

func (service *Service) Wait() {
	service.workers.Wait()
}

func (service *Service) run(id uuid.UUID, meterIds []uuid.UUID) {
	defer service.workers.Done()

	ctx, cancel := context.WithTimeout(service.lifetime, service.timeout)
	defer cancel()

	defer func() {
		if recovered := recover(); recovered != nil {
			service.fail(id, fmt.Errorf("panic during analysis: %v", recovered))
		}
	}()

	started := time.Now()

	if err := service.execute(ctx, id, meterIds); err != nil {
		service.fail(id, describeFailure(ctx, err, service.timeout))
		return
	}

	slog.Info("analysis completed", "analysis_id", id, "duration_ms", time.Since(started).Milliseconds())
}

func (service *Service) execute(ctx context.Context, id uuid.UUID, meterIds []uuid.UUID) error {
	if err := service.repository.Advance(ctx, id, stepLoading, progressLoading); err != nil {
		return err
	}

	dataset, err := service.repository.LoadDataset(ctx, meterIds)

	if err != nil {
		return err
	}

	if len(dataset.Meters) == 0 {
		return errors.New("there are no active meters to analyze")
	}

	if err := service.repository.Advance(ctx, id, stepAnalyzing, progressAnalyzing); err != nil {
		return err
	}

	result, err := service.analyzer.Analyze(ctx, service.buildRequest(dataset))

	if err != nil {
		return err
	}

	if err := service.repository.Advance(ctx, id, stepSaving, progressSaving); err != nil {
		return err
	}

	anomalies, summary, err := toAnomalies(id, result)

	if err != nil {
		return err
	}

	return service.repository.Complete(ctx, id, anomalies, summary)
}

func (service *Service) fail(id uuid.UUID, cause error) {
	slog.Error("analysis failed", "analysis_id", id, "error", cause)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(service.lifetime), failureWriteTimeout)
	defer cancel()

	if err := service.repository.Fail(ctx, id, cause.Error()); err != nil {
		slog.Error("could not record the analysis failure", "analysis_id", id, "error", err)
	}
}

func (service *Service) buildRequest(dataset Dataset) aiclient.AnalyzeRequest {
	request := aiclient.AnalyzeRequest{
		Timezone: service.timezone,
		Meters:   make([]aiclient.Meter, 0, len(dataset.Meters)),
		Readings: make([]aiclient.Reading, 0, len(dataset.Readings)),
		Events:   make([]aiclient.Event, 0, len(dataset.Events)),
	}

	for _, meter := range dataset.Meters {
		request.Meters = append(request.Meters, aiclient.Meter{
			Id:             meter.UidMeter.String(),
			Code:           meter.StrCodeMeter,
			NominalVoltage: meter.DecNominalVoltageMeter,
			MaxCurrent:     meter.DecMaxCurrentMeter,
		})
	}

	for _, reading := range dataset.Readings {
		request.Readings = append(request.Readings, aiclient.Reading{
			MeterId:        reading.UidMeter.String(),
			Timestamp:      reading.DtmTimestampReading,
			ConsumptionKwh: reading.DecConsumptionKwhReading,
			Voltage:        reading.DecVoltageReading,
			Current:        reading.DecCurrentReading,
			PowerFactor:    reading.DecPowerFactorReading,
		})
	}

	for _, event := range dataset.Events {
		request.Events = append(request.Events, aiclient.Event{
			Id:          event.UidEvent.String(),
			MeterId:     event.UidMeter.String(),
			Timestamp:   event.DtmTimestampEvent,
			Type:        string(event.StrTypeEvent),
			Description: event.StrDescriptionEvent,
		})
	}

	return request
}

type evidence struct {
	Signals          []aiclient.Signal          `json:"signals"`
	ChangedVariables []aiclient.ChangedVariable `json:"changedVariables"`
	RelatedEvent     *aiclient.EventReference   `json:"relatedEvent"`
}

func toAnomalies(analysisId uuid.UUID, result aiclient.AnalyzeResult) ([]db.InsertAnomalyParams, Summary, error) {
	anomalies := make([]db.InsertAnomalyParams, 0, len(result.Findings))
	summary := Summary{MetersAnalyzed: int32(result.MetersAnalyzed), Anomalies: int32(len(result.Findings))}

	for _, finding := range result.Findings {
		anomaly, err := toAnomaly(analysisId, finding)

		if err != nil {
			return nil, Summary{}, fmt.Errorf("invalid finding for meter %s: %w", finding.MeterCode, err)
		}

		if isHighPriority(anomaly) {
			summary.HighPriority++
		}

		anomalies = append(anomalies, anomaly)
	}

	return anomalies, summary, nil
}

func toAnomaly(analysisId uuid.UUID, finding aiclient.Finding) (db.InsertAnomalyParams, error) {
	meterId, err := uuid.Parse(finding.MeterId)

	if err != nil {
		return db.InsertAnomalyParams{}, fmt.Errorf("meter id %q is not a UUID", finding.MeterId)
	}

	anomalyType := db.AnomalyType(finding.Type)
	if !anomalyType.Valid() {
		return db.InsertAnomalyParams{}, fmt.Errorf("unknown anomaly type %q", finding.Type)
	}

	severity := db.AnomalySeverity(finding.Severity)
	if !severity.Valid() {
		return db.InsertAnomalyParams{}, fmt.Errorf("unknown severity %q", finding.Severity)
	}

	var eventId *uuid.UUID

	if finding.RelatedEvent != nil {
		parsed, err := uuid.Parse(finding.RelatedEvent.Id)

		if err != nil {
			return db.InsertAnomalyParams{}, fmt.Errorf("event id %q is not a UUID", finding.RelatedEvent.Id)
		}
		eventId = &parsed
	}

	evidenceJson, err := json.Marshal(evidence{
		Signals:          finding.Signals,
		ChangedVariables: finding.ChangedVariables,
		RelatedEvent:     finding.RelatedEvent,
	})

	if err != nil {
		return db.InsertAnomalyParams{}, fmt.Errorf("failed to encode evidence: %w", err)
	}

	changedNames := make([]string, 0, len(finding.ChangedVariables))
	for _, variable := range finding.ChangedVariables {
		changedNames = append(changedNames, variable.Name)
	}

	return db.InsertAnomalyParams{
		AnalysisID:        analysisId,
		MeterID:           meterId,
		EventID:           eventId,
		Type:              anomalyType,
		Severity:          severity,
		RuleID:            finding.RuleId,
		Confidence:        finding.Confidence,
		PriorityScore:     finding.PriorityScore,
		BaselineKwh:       &finding.BaselineKwh,
		CurrentKwh:        &finding.CurrentKwh,
		VariationPct:      &finding.VariationPct,
		WindowStart:       &finding.WindowStart,
		WindowEnd:         finding.WindowEnd,
		Reason:            finding.Reason,
		RecommendedAction: finding.RecommendedAction,
		ChangedVariables:  changedNames,
		Evidence:          evidenceJson,
		DetectedAt:        finding.DetectedAt,
	}, nil
}

func isHighPriority(anomaly db.InsertAnomalyParams) bool {
	return anomaly.Severity == db.AnomalySeverityHIGH && anomaly.Type != db.AnomalyTypeFALSEPOSITIVE
}

func parseMeterIds(raw []string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	meterIds := make([]uuid.UUID, 0, len(raw))

	for _, value := range raw {
		meterId, err := uuid.Parse(value)

		if err != nil {
			return nil, apperror.Validation(map[string]string{"meterIds": fmt.Sprintf("%q is not a valid UUID", value)})
		}
		meterIds = append(meterIds, meterId)
	}

	return meterIds, nil
}

func describeFailure(ctx context.Context, err error, timeout time.Duration) error {
	var apiErr *aiclient.APIError

	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return fmt.Errorf("the analysis exceeded the %s time limit", timeout)
	case errors.Is(ctx.Err(), context.Canceled):
		return errors.New("the analysis was interrupted because the service is shutting down")
	case errors.As(err, &apiErr):
		return fmt.Errorf("the AI service rejected the analysis (%s): %s", apiErr.Code, apiErr.Message)
	default:
		return err
	}
}
