package anomaly

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

const maxNoteLength = 500

var allowedTransitions = map[db.AnomalyStatus][]db.AnomalyStatus{
	db.AnomalyStatusOPEN:          {db.AnomalyStatusINVESTIGATING, db.AnomalyStatusRESOLVED, db.AnomalyStatusDISMISSED},
	db.AnomalyStatusINVESTIGATING: {db.AnomalyStatusOPEN, db.AnomalyStatusRESOLVED, db.AnomalyStatusDISMISSED},
	db.AnomalyStatusRESOLVED:      {db.AnomalyStatusOPEN},
	db.AnomalyStatusDISMISSED:     {db.AnomalyStatusOPEN},
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) List(ctx context.Context, query ListQuery) (ListResponse, error) {
	params, errs := toListParams(query)

	if len(errs) > 0 {
		return ListResponse{}, apperror.Validation(errs)
	}

	if query.AnalysisId == "" {
		latest, exists, err := service.repository.LatestCompletedAnalysisId(ctx)

		if err != nil {
			return ListResponse{}, err
		}

		if !exists {
			return ListResponse{Data: []SummaryResponse{}}, nil
		}
		params.AnalysisID = latest
	}

	rows, err := service.repository.List(ctx, params)

	if err != nil {
		return ListResponse{}, err
	}

	return ListResponse{AnalysisId: &params.AnalysisID, Data: toSummaryResponses(rows)}, nil
}

func (service *Service) Get(ctx context.Context, id uuid.UUID) (DetailResponse, error) {
	row, err := service.repository.GetById(ctx, id)

	if err != nil {
		return DetailResponse{}, mapError(err)
	}

	return toDetailResponse(row), nil
}

func (service *Service) UpdateStatus(ctx context.Context, id uuid.UUID, request UpdateStatusRequest) (DetailResponse, error) {
	status := db.AnomalyStatus(strings.ToUpper(strings.TrimSpace(request.Status)))
	note := normalizeNote(request.Note)
	errs := map[string]string{}

	if !status.Valid() {
		errs["status"] = "Must be one of OPEN, INVESTIGATING, RESOLVED, DISMISSED"
	}

	if note != nil && len(*note) > maxNoteLength {
		errs["note"] = fmt.Sprintf("Must be at most %d characters", maxNoteLength)
	}

	if len(errs) > 0 {
		return DetailResponse{}, apperror.Validation(errs)
	}

	current, err := service.repository.GetById(ctx, id)

	if err != nil {
		return DetailResponse{}, mapError(err)
	}

	if !slices.Contains(allowedTransitions[current.StrStatusAnomaly], status) {
		return DetailResponse{}, apperror.Conflict(
			"INVALID_STATUS_TRANSITION",
			fmt.Sprintf("An anomaly cannot move from %s to %s", current.StrStatusAnomaly, status),
		)
	}

	err = service.repository.UpdateStatus(ctx, db.UpdateAnomalyStatusParams{ID: id, Status: status, ResolutionNote: note})

	if err != nil {
		return DetailResponse{}, mapError(err)
	}

	return service.Get(ctx, id)
}

func toListParams(query ListQuery) (db.ListAnomaliesParams, map[string]string) {
	var params db.ListAnomaliesParams
	errs := map[string]string{}

	if query.AnalysisId != "" {
		analysisId, err := uuid.Parse(query.AnalysisId)

		if err != nil {
			errs["analysisId"] = "Must be a valid UUID"
		}
		params.AnalysisID = analysisId
	}

	if query.Type != "" {
		anomalyType := db.AnomalyType(strings.ToUpper(query.Type))

		if anomalyType.Valid() {
			params.Type = &anomalyType
		} else {
			errs["type"] = "Must be one of REAL_ANOMALY, EXPLAINABLE_ANOMALY, FALSE_POSITIVE, DATA_QUALITY"
		}
	}

	if query.Severity != "" {
		severity := db.AnomalySeverity(strings.ToUpper(query.Severity))

		if severity.Valid() {
			params.Severity = &severity
		} else {
			errs["severity"] = "Must be one of LOW, MEDIUM, HIGH"
		}
	}

	if query.Status != "" {
		status := db.AnomalyStatus(strings.ToUpper(query.Status))

		if status.Valid() {
			params.Status = &status
		} else {
			errs["status"] = "Must be one of OPEN, INVESTIGATING, RESOLVED, DISMISSED"
		}
	}

	return params, errs
}

func normalizeNote(note *string) *string {
	if note == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*note)

	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func mapError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("ANOMALY_NOT_FOUND", "Anomaly not found")
	}

	return err
}
