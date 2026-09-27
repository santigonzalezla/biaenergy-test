package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/aiclient"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type fakeRepository struct {
	mutex       sync.Mutex
	active      *db.Analysis
	latest      *db.Analysis
	stored      map[uuid.UUID]db.Analysis
	dataset     Dataset
	created     int
	steps       []string
	failures    []string
	anomalies   []db.InsertAnomalyParams
	summary     *Summary
	interrupted int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		stored: map[uuid.UUID]db.Analysis{},
		dataset: Dataset{
			Meters: []db.ListMetersForAnalysisRow{{UidMeter: uuid.New(), StrCodeMeter: "M-109", DecNominalVoltageMeter: 220}},
		},
	}
}

func (fake *fakeRepository) Create(ctx context.Context) (db.Analysis, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.created++
	analysis := db.Analysis{UidAnalysis: uuid.New(), StrStatusAnalysis: db.AnalysisStatusPENDING}
	fake.stored[analysis.UidAnalysis] = analysis

	return analysis, nil
}

func (fake *fakeRepository) GetById(ctx context.Context, id uuid.UUID) (db.Analysis, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	analysis, ok := fake.stored[id]
	if !ok {
		return db.Analysis{}, ErrNotFound
	}

	return analysis, nil
}

func (fake *fakeRepository) Latest(ctx context.Context) (db.Analysis, bool, error) {
	if fake.latest == nil {
		return db.Analysis{}, false, nil
	}

	return *fake.latest, true, nil
}

func (fake *fakeRepository) Active(ctx context.Context) (db.Analysis, bool, error) {
	if fake.active == nil {
		return db.Analysis{}, false, nil
	}

	return *fake.active, true, nil
}

func (fake *fakeRepository) Advance(ctx context.Context, id uuid.UUID, step string, progress int16) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.steps = append(fake.steps, step)

	return nil
}

func (fake *fakeRepository) Fail(ctx context.Context, id uuid.UUID, message string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.failures = append(fake.failures, message)

	return nil
}

func (fake *fakeRepository) FailInterrupted(ctx context.Context) (int64, error) {
	return fake.interrupted, nil
}

func (fake *fakeRepository) LoadDataset(ctx context.Context, meterIds []uuid.UUID) (Dataset, error) {
	return fake.dataset, nil
}

func (fake *fakeRepository) Complete(ctx context.Context, id uuid.UUID, anomalies []db.InsertAnomalyParams, summary Summary) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.anomalies = anomalies
	fake.summary = &summary

	return nil
}

type fakeAnalyzer struct {
	result   aiclient.AnalyzeResult
	err      error
	panics   bool
	blocks   bool
	requests []aiclient.AnalyzeRequest
}

func (fake *fakeAnalyzer) Analyze(ctx context.Context, request aiclient.AnalyzeRequest) (aiclient.AnalyzeResult, error) {
	fake.requests = append(fake.requests, request)

	if fake.panics {
		panic("unexpected nil pointer")
	}

	if fake.blocks {
		<-ctx.Done()
		return aiclient.AnalyzeResult{}, ctx.Err()
	}

	return fake.result, fake.err
}

func findings(meterId uuid.UUID, eventId uuid.UUID) aiclient.AnalyzeResult {
	start := time.Date(2026, 9, 12, 19, 0, 0, 0, time.UTC)

	return aiclient.AnalyzeResult{
		MetersAnalyzed: 12,
		Findings: []aiclient.Finding{
			{
				MeterId: meterId.String(), MeterCode: "M-109", Type: "REAL_ANOMALY", Severity: "HIGH",
				RuleId: "R4_UNEXPLAINED_SURGE", Confidence: 0.9, PriorityScore: 90,
				DetectedAt: start, WindowStart: start, BaselineKwh: 44.07, CurrentKwh: 92.77, VariationPct: 110.5,
				ChangedVariables: []aiclient.ChangedVariable{
					{Name: "consumption_kwh", Baseline: 42.38, Observed: 91.92, ChangePct: 116.9},
					{Name: "power_factor", Baseline: 0.94, Observed: 0.74, ChangePct: -21.3},
				},
				Signals:      []aiclient.Signal{{Kind: "CONSUMPTION_SURGE", StartedAt: start}},
				RelatedEvent: &aiclient.EventReference{Id: eventId.String(), Type: "UNKNOWN", Timestamp: start},
				Reason:       "El consumo aumentó.", RecommendedAction: "Revisar los equipos.",
			},
			{
				MeterId: meterId.String(), MeterCode: "M-106", Type: "FALSE_POSITIVE", Severity: "LOW",
				RuleId: "R2_SCHEDULED_OUTAGE", Confidence: 0.8, PriorityScore: 0.6,
				DetectedAt: start, WindowStart: start, Reason: "Corte programado.", RecommendedAction: "Ninguna.",
			},
		},
	}
}

func newService(repository *fakeRepository, analyzer *fakeAnalyzer) (*Service, context.CancelFunc) {
	lifetime, cancel := context.WithCancel(context.Background())

	return NewService(lifetime, repository, analyzer, "America/Bogota", time.Second), cancel
}

func TestStartRunsTheAnalysisInTheBackground(t *testing.T) {
	repository := newFakeRepository()
	eventId := uuid.New()
	analyzer := &fakeAnalyzer{result: findings(repository.dataset.Meters[0].UidMeter, eventId)}
	service, cancel := newService(repository, analyzer)
	defer cancel()

	response, created, err := service.Start(context.Background(), StartRequest{})
	service.Wait()

	if err != nil || !created || response.Status != db.AnalysisStatusPENDING {
		t.Fatalf("Start() = %+v, created=%v, err=%v", response, created, err)
	}

	if strings.Join(repository.steps, ",") != "loading_data,analyzing,saving" {
		t.Fatalf("steps = %v", repository.steps)
	}

	if len(repository.failures) != 0 {
		t.Fatalf("unexpected failures: %v", repository.failures)
	}

	if *repository.summary != (Summary{MetersAnalyzed: 12, Anomalies: 2, HighPriority: 1}) {
		t.Fatalf("summary = %+v, want 12 meters, 2 anomalies, 1 high priority", *repository.summary)
	}

	request := analyzer.requests[0]
	if request.Timezone != "America/Bogota" || request.Meters[0].Code != "M-109" {
		t.Fatalf("request sent to the AI = %+v", request)
	}
}

func TestFindingsAreStoredWithTheirEvidence(t *testing.T) {
	repository := newFakeRepository()
	eventId := uuid.New()
	service, cancel := newService(repository, &fakeAnalyzer{result: findings(repository.dataset.Meters[0].UidMeter, eventId)})
	defer cancel()

	_, _, _ = service.Start(context.Background(), StartRequest{})
	service.Wait()

	surge := repository.anomalies[0]
	if surge.Type != db.AnomalyTypeREALANOMALY || surge.Severity != db.AnomalySeverityHIGH || surge.RuleID != "R4_UNEXPLAINED_SURGE" {
		t.Fatalf("anomaly = %+v", surge)
	}

	if surge.EventID == nil || *surge.EventID != eventId {
		t.Fatalf("event id = %v, want %v", surge.EventID, eventId)
	}

	if strings.Join(surge.ChangedVariables, ",") != "consumption_kwh,power_factor" {
		t.Fatalf("changed variables = %v", surge.ChangedVariables)
	}

	var stored evidence
	if err := json.Unmarshal(surge.Evidence, &stored); err != nil {
		t.Fatalf("evidence is not valid JSON: %v", err)
	}
	if len(stored.ChangedVariables) != 2 || stored.ChangedVariables[1].Observed != 0.74 || stored.RelatedEvent == nil {
		t.Fatalf("evidence = %+v", stored)
	}

	outage := repository.anomalies[1]
	if outage.EventID != nil || outage.WindowEnd != nil {
		t.Fatalf("outage without event must keep nil references: %+v", outage)
	}
}

func TestStartReusesTheAnalysisInProgress(t *testing.T) {
	repository := newFakeRepository()
	running := db.Analysis{UidAnalysis: uuid.New(), StrStatusAnalysis: db.AnalysisStatusRUNNING}
	repository.active = &running
	analyzer := &fakeAnalyzer{}
	service, cancel := newService(repository, analyzer)
	defer cancel()

	response, created, err := service.Start(context.Background(), StartRequest{})
	service.Wait()

	if err != nil || created || response.Id != running.UidAnalysis {
		t.Fatalf("Start() = %+v, created=%v, err=%v, want the running analysis", response, created, err)
	}

	if repository.created != 0 || len(analyzer.requests) != 0 {
		t.Fatal("a second analysis must not be created nor sent to the AI")
	}
}

func TestFailures(t *testing.T) {
	tests := []struct {
		name        string
		analyzer    *fakeAnalyzer
		noMeters    bool
		cancelEarly bool
		wantMessage string
	}{
		{
			name:        "AI service rejects the request",
			analyzer:    &fakeAnalyzer{err: &aiclient.APIError{StatusCode: http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: "invalid data"}},
			wantMessage: "VALIDATION_ERROR",
		},
		{
			name:        "AI returns an unknown anomaly type",
			analyzer:    &fakeAnalyzer{result: aiclient.AnalyzeResult{Findings: []aiclient.Finding{{MeterId: uuid.NewString(), MeterCode: "M-1", Type: "WEIRD", Severity: "HIGH"}}}},
			wantMessage: "unknown anomaly type",
		},
		{
			name:        "No active meters",
			analyzer:    &fakeAnalyzer{},
			noMeters:    true,
			wantMessage: "no active meters",
		},
		{
			name:        "AI takes longer than the timeout",
			analyzer:    &fakeAnalyzer{blocks: true},
			wantMessage: "time limit",
		},
		{
			name:        "Backend shuts down during the analysis",
			analyzer:    &fakeAnalyzer{blocks: true},
			cancelEarly: true,
			wantMessage: "shutting down",
		},
		{
			name:        "Panic inside the worker",
			analyzer:    &fakeAnalyzer{panics: true},
			wantMessage: "panic",
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := newFakeRepository()
			if tableTest.noMeters {
				repository.dataset = Dataset{}
			}

			lifetime, cancel := context.WithCancel(context.Background())
			defer cancel()
			service := NewService(lifetime, repository, tableTest.analyzer, "America/Bogota", 100*time.Millisecond)

			if _, _, err := service.Start(context.Background(), StartRequest{}); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			if tableTest.cancelEarly {
				time.Sleep(20 * time.Millisecond)
				cancel()
			}
			service.Wait()

			if len(repository.failures) != 1 || !strings.Contains(repository.failures[0], tableTest.wantMessage) {
				t.Fatalf("failures = %v, want one containing %q", repository.failures, tableTest.wantMessage)
			}
			if repository.summary != nil {
				t.Fatal("a failed analysis must not be completed")
			}
		})
	}
}

func TestStartValidatesMeterIds(t *testing.T) {
	service, cancel := newService(newFakeRepository(), &fakeAnalyzer{})
	defer cancel()

	_, _, err := service.Start(context.Background(), StartRequest{MeterIds: []string{"not-a-uuid"}})

	if status := apperror.From(err).Status; status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
}

func TestQueriesReturnNotFound(t *testing.T) {
	service, cancel := newService(newFakeRepository(), &fakeAnalyzer{})
	defer cancel()

	_, getErr := service.Get(context.Background(), uuid.New())
	_, latestErr := service.Latest(context.Background())

	if code := apperror.From(getErr).Code; code != "ANALYSIS_NOT_FOUND" {
		t.Fatalf("Get code = %q", code)
	}
	if code := apperror.From(latestErr).Code; code != "NO_ANALYSIS_YET" {
		t.Fatalf("Latest code = %q", code)
	}
}

func TestRecoverInterruptedDoesNotFail(t *testing.T) {
	repository := newFakeRepository()
	repository.interrupted = 2
	service, cancel := newService(repository, &fakeAnalyzer{})
	defer cancel()

	if err := service.RecoverInterrupted(context.Background()); err != nil {
		t.Fatalf("RecoverInterrupted() error = %v", err)
	}
}

func TestDescribeFailureKeepsUnknownErrors(t *testing.T) {
	cause := errors.New("database is down")

	if got := describeFailure(context.Background(), cause, time.Minute); !errors.Is(got, cause) {
		t.Fatalf("describeFailure() = %v, want the original error", got)
	}
}
