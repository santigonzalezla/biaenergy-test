package meter

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/pagination"
)

const (
	defaultNominalVoltage = 220
	defaultSector         = "INDUSTRIAL"
	defaultSortBy         = "code"
	defaultSeriesWindow   = 7 * 24 * time.Hour
	maxSeriesWindow       = 31 * 24 * time.Hour
	statsBaselineDays     = 7
	statsRecentHours      = 48
)

const (
	anchorLatest   rangeAnchor = iota // 0
	anchorEarliest                    //1
)

var (
	statsFrom = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	statsTo   = time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
)

var sorteableFields = map[string]bool{
	"code":   true,
	"name":   true,
	"status": true,
}

type rangeAnchor int

type ListQuery struct {
	Page     int
	Limit    int
	Status   string
	Search   string
	SortBy   string
	SortDesc bool
}

type SeriesQuery struct {
	From *time.Time
	To   *time.Time
}

type Service struct {
	repository Repository
	location   *time.Location
}

func NewService(repository Repository, location *time.Location) *Service {
	return &Service{repository: repository, location: location}
}

func (service *Service) List(ctx context.Context, query ListQuery) (pagination.Page[MeterResponse], error) {
	page, limit, offset := pagination.Normalize(query.Page, query.Limit)
	errs := map[string]string{}

	params := db.ListMetersParams{
		SortBy:     defaultSortBy,
		SortDesc:   query.SortDesc,
		PageLimit:  int32(limit),
		PageOffset: int32(offset),
	}

	if query.SortBy != "" {
		if sorteableFields[query.SortBy] {
			params.SortBy = query.SortBy
		} else {
			errs["sortBy"] = "Must be between code, name or status"
		}
	}

	if query.Status != "" {
		status := db.MeterStatus(strings.ToUpper(query.Status))

		if status.Valid() {
			params.Status = &status
		} else {
			errs["status"] = "Must be between OK, ALERT, CRITICAL"
		}
	}

	if search := strings.TrimSpace(query.Search); search != "" {
		params.Search = &search
	}

	if len(errs) > 0 {
		return pagination.Page[MeterResponse]{}, apperror.Validation(errs)
	}

	meters, total, err := service.repository.List(ctx, params)

	if err != nil {
		return pagination.Page[MeterResponse]{}, err
	}

	responses := toMeterResponses(meters)

	if err := service.attachStats(ctx, responses); err != nil {
		return pagination.Page[MeterResponse]{}, err
	}

	return pagination.Page[MeterResponse]{
		Data:  responses,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (service *Service) Get(ctx context.Context, id uuid.UUID) (MeterResponse, error) {
	meter, err := service.repository.GetById(ctx, id)

	if err != nil {
		return MeterResponse{}, mapError(err)
	}

	return toMeterResponse(meter), nil
}

func (service *Service) Create(ctx context.Context, request CreateMeterRequest) (MeterResponse, error) {
	if errs := request.Validate(); len(errs) > 0 {
		return MeterResponse{}, apperror.Validation(errs)
	}

	params := db.CreateMeterParams{
		Code:              strings.ToUpper(strings.TrimSpace(request.Code)),
		Name:              strings.TrimSpace(request.Name),
		Location:          strings.TrimSpace(request.Location),
		Sector:            defaultSector,
		NominalVoltage:    defaultNominalVoltage,
		MaxCurrent:        request.MaxCurrent,
		ContractedPowerKw: request.ContractedPowerKw,
	}

	if sector := strings.TrimSpace(request.Sector); sector != "" {
		params.Sector = strings.ToUpper(sector)
	}

	if request.NominalVoltage != nil {
		params.NominalVoltage = *request.NominalVoltage
	}

	meter, err := service.repository.Create(ctx, params)

	if err != nil {
		return MeterResponse{}, mapError(err)
	}

	return toMeterResponse(meter), nil
}

func (service *Service) Update(ctx context.Context, id uuid.UUID, request UpdateMeterRequest) (MeterResponse, error) {
	if errs := request.Validate(); len(errs) > 0 {
		return MeterResponse{}, apperror.Validation(errs)
	}

	params := db.UpdateMeterParams{
		ID:                id,
		Name:              trimmed(request.Name),
		Location:          trimmed(request.Location),
		Sector:            trimmed(request.Sector),
		NominalVoltage:    request.NominalVoltage,
		MaxCurrent:        request.MaxCurrent,
		ContractedPowerKw: request.ContractedPowerKw,
	}

	if request.Status != nil {
		status := db.MeterStatus(*request.Status)
		params.Status = &status
	}

	meter, err := service.repository.Update(ctx, params)

	if err != nil {
		return MeterResponse{}, mapError(err)
	}

	return toMeterResponse(meter), nil
}

func (service *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return mapError(service.repository.SoftDelete(ctx, id))
}

func (service *Service) ListReadings(ctx context.Context, id uuid.UUID, query SeriesQuery) (SeriesResponse[ReadingResponse], error) {
	from, to, err := service.resolveRange(ctx, id, query, anchorLatest)

	if err != nil {
		return SeriesResponse[ReadingResponse]{}, err
	}

	rows, err := service.repository.ListReadings(ctx, db.ListReadingsByMeterParams{
		MeterID:  id,
		FromTime: from,
		ToTime:   to,
	})

	if err != nil {
		return SeriesResponse[ReadingResponse]{}, err
	}

	return SeriesResponse[ReadingResponse]{
		From: from.UTC(),
		To:   to.UTC(),
		Data: toReadingResponses(rows),
	}, nil
}

func (service *Service) ListEvents(ctx context.Context, id uuid.UUID, query SeriesQuery) (SeriesResponse[EventResponse], error) {
	from, to, err := service.resolveRange(ctx, id, query, anchorLatest)

	if err != nil {
		return SeriesResponse[EventResponse]{}, err
	}

	rows, err := service.repository.ListEvents(ctx, db.ListEventsByMeterParams{
		MeterID:  id,
		FromTime: from,
		ToTime:   to,
	})

	if err != nil {
		return SeriesResponse[EventResponse]{}, err
	}

	return SeriesResponse[EventResponse]{
		From: from.UTC(),
		To:   to.UTC(),
		Data: toEventResponses(rows),
	}, nil
}

func (service *Service) HourlyProfile(ctx context.Context, id uuid.UUID, query SeriesQuery) (ProfileResponse, error) {
	from, to, err := service.resolveRange(ctx, id, query, anchorEarliest)

	if err != nil {
		return ProfileResponse{}, err
	}

	rows, err := service.repository.HourlyProfile(ctx, db.GetMeterHourlyProfileParams{
		MeterID:  id,
		FromTime: from,
		ToTime:   to,
		Timezone: service.location.String(),
	})

	if err != nil {
		return ProfileResponse{}, err
	}

	return ProfileResponse{
		From:     from.UTC(),
		To:       to.UTC(),
		Timezone: service.location.String(),
		Hours:    toHourlyValues(rows),
	}, nil
}

func (service *Service) resolveRange(ctx context.Context, id uuid.UUID, query SeriesQuery, anchor rangeAnchor) (time.Time, time.Time, error) {
	if _, err := service.repository.GetById(ctx, id); err != nil {
		return time.Time{}, time.Time{}, mapError(err)
	}

	var from, to time.Time

	switch {
	case query.From != nil && query.To != nil:
		from, to = *query.From, *query.To
	case query.From != nil:
		from = *query.From
		to = from.Add(defaultSeriesWindow)
	case query.To != nil:
		to = *query.To
		from = to.Add(-defaultSeriesWindow)
	default:
		var err error
		from, to, err = service.defaultRange(ctx, id, anchor)

		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	errs := map[string]string{}

	if !from.Before(to) {
		errs["from"] = "must be before to"
	} else if to.Sub(from) > maxSeriesWindow {
		errs["to"] = "range must be 31 days max"
	}

	if len(errs) > 0 {
		return time.Time{}, time.Time{}, apperror.Validation(errs)
	}

	return from, to, nil
}

func (service *Service) attachStats(ctx context.Context, responses []MeterResponse) error {
	if len(responses) == 0 {
		return nil
	}

	rows, err := service.repository.ConsumptionStats(ctx, db.GetMeterConsumptionStatsParams{
		FromTime:     statsFrom,
		ToTime:       statsTo,
		BaselineDays: statsBaselineDays,
		RecentHours:  statsRecentHours,
	})

	if err != nil {
		return err
	}

	statsByMeter := make(map[uuid.UUID]*StatsResponse, len(rows))

	for _, row := range rows {
		statsByMeter[row.UidMeter] = toStatsResponse(row)
	}

	for i := range responses {
		responses[i].Stats = statsByMeter[responses[i].ID]
	}

	return nil
}

func (service *Service) defaultRange(ctx context.Context, id uuid.UUID, anchor rangeAnchor) (time.Time, time.Time,
	error) {
	if anchor == anchorEarliest {
		earliest, ok, err := service.repository.EarliestReadingTime(ctx, id)

		if err != nil {
			return time.Time{}, time.Time{}, err
		}

		if ok {
			from := earliest.Truncate(time.Hour)

			return from, from.Add(defaultSeriesWindow), nil
		}
	} else {
		latest, ok, err := service.repository.LatestReadingTime(ctx, id)

		if err != nil {
			return time.Time{}, time.Time{}, err
		}

		if ok {
			to := latest.Truncate(time.Hour).Add(time.Hour)

			return to.Add(-defaultSeriesWindow), to, nil
		}
	}

	to := time.Now().Truncate(time.Hour).Add(time.Hour)

	return to.Add(-defaultSeriesWindow), to, nil
}

func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound):
		return apperror.NotFound("METER_NOT_FOUND", "Meter not found")
	case errors.Is(err, ErrDuplicateCode):
		return apperror.Conflict("METER_DUPLICATE_CODE", "Meter code already exists")
	default:
		return err
	}
}

func trimmed(value *string) *string {
	if value == nil {
		return nil
	}

	result := strings.TrimSpace(*value)

	return &result
}
