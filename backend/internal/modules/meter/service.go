package meter

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/pagination"
)

const (
	defaultNominalVoltage = 220
	defaultSector         = "INDUSTRIAL"
	defaultSortBy         = "code"
)

var sorteableFields = map[string]bool{
	"code":   true,
	"name":   true,
	"status": true,
}

type ListQuery struct {
	Page     int
	Limit    int
	Status   string
	Search   string
	SortBy   string
	SortDesc bool
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
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

	return pagination.Page[MeterResponse]{
		Data:  toMeterResponses(meters),
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
