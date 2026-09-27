package anomaly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
)

type fakeRepository struct {
	latestAnalysis *uuid.UUID
	rows           map[uuid.UUID]db.GetAnomalyRow
	listParams     *db.ListAnomaliesParams
	updates        []db.UpdateAnomalyStatusParams
}

func newFakeRepository(anomalies ...db.GetAnomalyRow) *fakeRepository {
	fake := &fakeRepository{rows: map[uuid.UUID]db.GetAnomalyRow{}}
	for _, anomaly := range anomalies {
		fake.rows[anomaly.UidAnomaly] = anomaly
	}
	return fake
}

func (fake *fakeRepository) LatestCompletedAnalysisId(ctx context.Context) (uuid.UUID, bool, error) {
	if fake.latestAnalysis == nil {
		return uuid.Nil, false, nil
	}
	return *fake.latestAnalysis, true, nil
}

func (fake *fakeRepository) List(ctx context.Context, params db.ListAnomaliesParams) ([]db.ListAnomaliesRow, error) {
	fake.listParams = &params
	return []db.ListAnomaliesRow{{UidAnomaly: uuid.New(), StrCodeMeter: "M-109", StrTypeAnomaly: db.AnomalyTypeREALANOMALY}}, nil
}

func (fake *fakeRepository) GetById(ctx context.Context, id uuid.UUID) (db.GetAnomalyRow, error) {
	row, ok := fake.rows[id]
	if !ok {
		return db.GetAnomalyRow{}, ErrNotFound
	}
	return row, nil
}

func (fake *fakeRepository) UpdateStatus(ctx context.Context, params db.UpdateAnomalyStatusParams) error {
	fake.updates = append(fake.updates, params)
	row := fake.rows[params.ID]
	row.StrStatusAnomaly = params.Status
	if params.ResolutionNote != nil {
		row.StrResolutionNoteAnomaly = params.ResolutionNote
	}
	fake.rows[params.ID] = row
	return nil
}

func anomalyRow(status db.AnomalyStatus) db.GetAnomalyRow {
	eventId := uuid.New()
	eventType := db.EventTypeUNKNOWN
	eventAt := time.Date(2026, 9, 12, 19, 0, 0, 0, time.UTC)
	description := "No operational event reported"

	return db.GetAnomalyRow{
		UidAnomaly:          uuid.New(),
		UidAnalysis:         uuid.New(),
		UidMeter:            uuid.New(),
		StrCodeMeter:        "M-109",
		StrTypeAnomaly:      db.AnomalyTypeREALANOMALY,
		StrSeverityAnomaly:  db.AnomalySeverityHIGH,
		StrStatusAnomaly:    status,
		StrRuleIDAnomaly:    "R4_UNEXPLAINED_SURGE",
		JsonEvidenceAnomaly: json.RawMessage(`{"signals": []}`),
		UidEvent:            &eventId,
		StrTypeEvent:        &eventType,
		DtmTimestampEvent:   &eventAt,
		StrDescriptionEvent: &description,
	}
}

func TestListUsesTheLatestCompletedAnalysisByDefault(t *testing.T) {
	latest := uuid.New()
	repository := newFakeRepository()
	repository.latestAnalysis = &latest

	response, err := NewService(repository).List(context.Background(), ListQuery{Type: "real_anomaly", Status: "open"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if *response.AnalysisId != latest || repository.listParams.AnalysisID != latest {
		t.Fatalf("analysis = %v, want the latest completed %v", response.AnalysisId, latest)
	}
	if *repository.listParams.Type != db.AnomalyTypeREALANOMALY || *repository.listParams.Status != db.AnomalyStatusOPEN {
		t.Fatalf("filters = %+v", repository.listParams)
	}
}

func TestListWithoutAnyCompletedAnalysisIsEmpty(t *testing.T) {
	repository := newFakeRepository()

	response, err := NewService(repository).List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if response.AnalysisId != nil || response.Data == nil || len(response.Data) != 0 {
		t.Fatalf("response = %+v, want no analysis and an empty list", response)
	}
	if repository.listParams != nil {
		t.Fatal("the repository must not be queried without an analysis")
	}
}

func TestListValidatesFilters(t *testing.T) {
	_, err := NewService(newFakeRepository()).List(context.Background(), ListQuery{AnalysisId: "abc", Severity: "EXTREME"})

	details, _ := apperror.From(err).Details.(map[string]string)
	if apperror.From(err).Status != http.StatusBadRequest || details["analysisId"] == "" || details["severity"] == "" {
		t.Fatalf("error = %+v, want both fields reported", apperror.From(err))
	}
}

func TestUpdateStatusTransitions(t *testing.T) {
	tests := []struct {
		name       string
		from       db.AnomalyStatus
		to         string
		wantStatus int
		wantCode   string
	}{
		{name: "Start investigating", from: db.AnomalyStatusOPEN, to: "INVESTIGATING"},
		{name: "Resolve after investigating", from: db.AnomalyStatusINVESTIGATING, to: "resolved"},
		{name: "Dismiss directly", from: db.AnomalyStatusOPEN, to: "DISMISSED"},
		{name: "Reopen a resolved anomaly", from: db.AnomalyStatusRESOLVED, to: "OPEN"},
		{name: "Resolved cannot jump to investigating", from: db.AnomalyStatusRESOLVED, to: "INVESTIGATING", wantStatus: http.StatusConflict, wantCode: "INVALID_STATUS_TRANSITION"},
		{name: "Same status is not a transition", from: db.AnomalyStatusOPEN, to: "OPEN", wantStatus: http.StatusConflict, wantCode: "INVALID_STATUS_TRANSITION"},
		{name: "Unknown status", from: db.AnomalyStatusOPEN, to: "FIXED", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			row := anomalyRow(tableTest.from)
			repository := newFakeRepository(row)

			response, err := NewService(repository).UpdateStatus(context.Background(), row.UidAnomaly, UpdateStatusRequest{Status: tableTest.to})

			if tableTest.wantStatus != 0 {
				appErr := apperror.From(err)
				if appErr.Status != tableTest.wantStatus || appErr.Code != tableTest.wantCode {
					t.Fatalf("error = %d %s, want %d %s", appErr.Status, appErr.Code, tableTest.wantStatus, tableTest.wantCode)
				}
				if len(repository.updates) != 0 {
					t.Fatal("an invalid change must not reach the repository")
				}
				return
			}

			if err != nil {
				t.Fatalf("UpdateStatus() error = %v", err)
			}
			if string(response.Status) != strings.ToUpper(tableTest.to) {
				t.Fatalf("status = %s, want %s", response.Status, tableTest.to)
			}
		})
	}
}

func TestUpdateStatusStoresATrimmedNote(t *testing.T) {
	row := anomalyRow(db.AnomalyStatusOPEN)
	repository := newFakeRepository(row)
	note := "  Compresor 3 con rodamiento dañado  "

	response, err := NewService(repository).UpdateStatus(context.Background(), row.UidAnomaly, UpdateStatusRequest{Status: "RESOLVED", Note: &note})
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	if response.ResolutionNote == nil || *response.ResolutionNote != "Compresor 3 con rodamiento dañado" {
		t.Fatalf("note = %v", response.ResolutionNote)
	}
}

func TestUpdateStatusRejectsLongNotes(t *testing.T) {
	row := anomalyRow(db.AnomalyStatusOPEN)
	note := strings.Repeat("a", maxNoteLength+1)

	_, err := NewService(newFakeRepository(row)).UpdateStatus(context.Background(), row.UidAnomaly, UpdateStatusRequest{Status: "RESOLVED", Note: &note})

	if apperror.From(err).Status != http.StatusBadRequest {
		t.Fatalf("error = %v, want 400", err)
	}
}

func TestDetailIncludesMeterEventAndEvidence(t *testing.T) {
	row := anomalyRow(db.AnomalyStatusOPEN)

	detail, err := NewService(newFakeRepository(row)).Get(context.Background(), row.UidAnomaly)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if detail.Meter.Code != "M-109" || detail.RuleId != "R4_UNEXPLAINED_SURGE" {
		t.Fatalf("detail = %+v", detail)
	}
	if detail.RelatedEvent == nil || detail.RelatedEvent.Description != "No operational event reported" {
		t.Fatalf("related event = %+v", detail.RelatedEvent)
	}
	if string(detail.Evidence) != `{"signals": []}` {
		t.Fatalf("evidence = %s", detail.Evidence)
	}
}

func TestHandler(t *testing.T) {
	row := anomalyRow(db.AnomalyStatusOPEN)
	latest := uuid.New()

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "List", method: http.MethodGet, path: "/anomalies?severity=high", wantStatus: http.StatusOK},
		{name: "Detail", method: http.MethodGet, path: "/anomalies/" + row.UidAnomaly.String(), wantStatus: http.StatusOK},
		{name: "Unknown anomaly", method: http.MethodGet, path: "/anomalies/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantCode: "ANOMALY_NOT_FOUND"},
		{name: "Invalid id", method: http.MethodGet, path: "/anomalies/abc", wantStatus: http.StatusBadRequest, wantCode: "INVALID_ANOMALY_ID"},
		{name: "Operator action", method: http.MethodPatch, path: "/anomalies/" + row.UidAnomaly.String() + "/status", body: `{"status":"INVESTIGATING","note":"Revisando compresores"}`, wantStatus: http.StatusOK},
		{name: "Action with unknown field", method: http.MethodPatch, path: "/anomalies/" + row.UidAnomaly.String() + "/status", body: `{"status":"RESOLVED","priority":1}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY"},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := newFakeRepository(row)
			repository.latestAnalysis = &latest

			router := chi.NewRouter()
			NewHandler(NewService(repository)).RegisterRoutes(router)

			request := httptest.NewRequest(tableTest.method, tableTest.path, strings.NewReader(tableTest.body))
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != tableTest.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, tableTest.wantStatus, recorder.Body.String())
			}

			if tableTest.wantCode != "" {
				var body httpserver.ErrorBody
				if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
					t.Fatalf("invalid error body: %v", err)
				}
				if body.Error.Code != tableTest.wantCode {
					t.Fatalf("code = %q, want %q", body.Error.Code, tableTest.wantCode)
				}
			}
		})
	}
}
