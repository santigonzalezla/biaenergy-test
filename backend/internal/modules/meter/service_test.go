package meter

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type fakeRepository struct {
	err          error
	meter        db.Meter
	called       bool
	listParams   db.ListMetersParams
	createParams db.CreateMeterParams
	updateParams db.UpdateMeterParams
}

func (fake *fakeRepository) List(ctx context.Context, params db.ListMetersParams) ([]db.Meter, int64, error) {
	fake.called = true
	fake.listParams = params

	return []db.Meter{fake.meter}, 1, fake.err
}

func (fake *fakeRepository) GetById(ctx context.Context, id uuid.UUID) (db.Meter, error) {
	fake.called = true

	return fake.meter, fake.err
}

func (fake *fakeRepository) Create(ctx context.Context, params db.CreateMeterParams) (db.Meter, error) {
	fake.called = true
	fake.createParams = params

	return fake.meter, fake.err
}

func (fake *fakeRepository) Update(ctx context.Context, params db.UpdateMeterParams) (db.Meter, error) {
	fake.called = true
	fake.updateParams = params

	return fake.meter, fake.err
}

func (fake *fakeRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	fake.called = true

	return fake.err
}

func assertStatus(t *testing.T, err error, wantStatus int) {
	t.Helper()

	if wantStatus == 0 {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}

	if err == nil {
		t.Fatalf("expected error with status %d, got nil", wantStatus)
	}

	if got := apperror.From(err).Status; got != wantStatus {
		t.Fatalf("status = %d, want %d (err: %v)", got, wantStatus, err)
	}
}

func float(value float64) *float64 {
	return &value
}

func text(value string) *string {
	return &value
}

func TestServiceList(t *testing.T) {
	tests := []struct {
		name       string
		query      ListQuery
		wantStatus int
		check      func(t *testing.T, params db.ListMetersParams)
	}{
		{
			name:  "Defaults",
			query: ListQuery{},
			check: func(t *testing.T, params db.ListMetersParams) {
				if params.SortBy != "code" || params.PageLimit != 20 || params.PageOffset != 0 {
					t.Fatalf("unexpected params: %+v", params)
				}
				if params.Status != nil || params.Search != nil {
					t.Fatalf("filters must be nil when not sent: %+v", params)
				}
			},
		},
		{
			name:  "Pagination and filters",
			query: ListQuery{Page: 3, Limit: 10, Status: "alert", Search: "  planta ", SortBy: "name", SortDesc: true},
			check: func(t *testing.T, params db.ListMetersParams) {
				if params.PageOffset != 20 || params.PageLimit != 10 {
					t.Fatalf("offset/limit = %d/%d, want 20/10", params.PageOffset, params.PageLimit)
				}
				if params.Status == nil || *params.Status != db.MeterStatusALERT {
					t.Fatalf("status = %v, want ALERT", params.Status)
				}
				if params.Search == nil || *params.Search != "planta" {
					t.Fatalf("search = %v, want planta", params.Search)
				}
				if params.SortBy != "name" || !params.SortDesc {
					t.Fatalf("sort = %s desc=%v, want name desc", params.SortBy, params.SortDesc)
				}
			},
		},
		{name: "Invalid status", query: ListQuery{Status: "BROKEN"}, wantStatus: http.StatusBadRequest},
		{name: "Invalid sort field", query: ListQuery{SortBy: "password"}, wantStatus: http.StatusBadRequest},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)

			_, err := service.List(context.Background(), tableTest.query)

			assertStatus(t, err, tableTest.wantStatus)

			if tableTest.check != nil {
				tableTest.check(t, repository.listParams)
			}
		})
	}
}

func TestServiceCreate(t *testing.T) {
	tests := []struct {
		name           string
		request        CreateMeterRequest
		repositoryErr  error
		wantStatus     int
		wantRepoCalled bool
		check          func(t *testing.T, params db.CreateMeterParams)
	}{
		{
			name:           "Applies defaults and normalizes",
			request:        CreateMeterRequest{Code: " m-113 ", Name: " Compresor 3 "},
			wantRepoCalled: true,
			check: func(t *testing.T, params db.CreateMeterParams) {
				if params.Code != "M-113" || params.Name != "Compresor 3" {
					t.Fatalf("code/name = %q/%q", params.Code, params.Name)
				}
				if params.NominalVoltage != 220 || params.Sector != "INDUSTRIAL" {
					t.Fatalf("defaults not applied: %+v", params)
				}
			},
		},
		{
			name:           "Keeps explicit voltage",
			request:        CreateMeterRequest{Code: "M-114", Name: "Línea 2", NominalVoltage: float(440)},
			wantRepoCalled: true,
			check: func(t *testing.T, params db.CreateMeterParams) {
				if params.NominalVoltage != 440 {
					t.Fatalf("voltage = %v, want 440", params.NominalVoltage)
				}
			},
		},
		{
			name:       "Invalid request never reaches repository",
			request:    CreateMeterRequest{Code: "?", Name: "", NominalVoltage: float(-1)},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:           "Duplicate code is 409",
			request:        CreateMeterRequest{Code: "M-101", Name: "Planta"},
			repositoryErr:  ErrDuplicateCode,
			wantStatus:     http.StatusConflict,
			wantRepoCalled: true,
		},
		{
			name:       "Empty name is rejected",
			request:    CreateMeterRequest{Code: "M-115", Name: "   "},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{err: tableTest.repositoryErr}
			service := NewService(repository)

			_, err := service.Create(context.Background(), tableTest.request)

			assertStatus(t, err, tableTest.wantStatus)

			if repository.called != tableTest.wantRepoCalled {
				t.Fatalf("repository called = %v, want %v", repository.called, tableTest.wantRepoCalled)
			}

			if tableTest.check != nil {
				tableTest.check(t, repository.createParams)
			}
		})
	}
}

func TestServiceUpdate(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	_, err := service.Update(context.Background(), uuid.New(), UpdateMeterRequest{
		Name:   text("  Nuevo nombre "),
		Status: text("CRITICAL"),
	})

	assertStatus(t, err, 0)

	params := repository.updateParams

	if params.Name == nil || *params.Name != "Nuevo nombre" {
		t.Fatalf("name = %v, want trimmed", params.Name)
	}

	if params.Status == nil || *params.Status != db.MeterStatusCRITICAL {
		t.Fatalf("status = %v, want CRITICAL", params.Status)
	}

	if params.Location != nil || params.NominalVoltage != nil {
		t.Fatalf("fields not sent must stay nil: %+v", params)
	}
}

func TestServiceErrorMapping(t *testing.T) {
	tests := []struct {
		name          string
		repositoryErr error
		wantStatus    int
	}{
		{name: "Not found is 404", repositoryErr: ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "Unknown error is 500", repositoryErr: errors.New("connection refused"), wantStatus: http.StatusInternalServerError},
		{name: "No error", repositoryErr: nil, wantStatus: 0},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			service := NewService(&fakeRepository{err: tableTest.repositoryErr})

			_, getErr := service.Get(context.Background(), uuid.New())
			assertStatus(t, getErr, tableTest.wantStatus)

			deleteErr := service.Delete(context.Background(), uuid.New())
			assertStatus(t, deleteErr, tableTest.wantStatus)
		})
	}
}
