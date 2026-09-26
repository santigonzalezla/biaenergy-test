package meter

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type fakeRepository struct {
	err            error
	meter          db.Meter
	called         bool
	listParams     db.ListMetersParams
	createParams   db.CreateMeterParams
	updateParams   db.UpdateMeterParams
	readings       []db.ListReadingsByMeterRow
	events         []db.ListEventsByMeterRow
	latestReading  time.Time
	hasReadings    bool
	readingsParams db.ListReadingsByMeterParams
	eventsParams   db.ListEventsByMeterParams
	stats          []db.GetMeterConsumptionStatsRow
	statsErr       error
	earliest       time.Time
	profile        []db.GetMeterHourlyProfileRow
	profileParams  db.GetMeterHourlyProfileParams
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

func (fake *fakeRepository) ListReadings(ctx context.Context, params db.ListReadingsByMeterParams) ([]db.ListReadingsByMeterRow, error) {
	fake.readingsParams = params

	return fake.readings, nil
}

func (fake *fakeRepository) ListEvents(ctx context.Context, params db.ListEventsByMeterParams) ([]db.ListEventsByMeterRow, error) {
	fake.eventsParams = params

	return fake.events, nil
}

func (fake *fakeRepository) LatestReadingTime(ctx context.Context, id uuid.UUID) (time.Time, bool, error) {
	return fake.latestReading, fake.hasReadings, nil
}

func (fake *fakeRepository) ConsumptionStats(ctx context.Context, params db.GetMeterConsumptionStatsParams) ([]db.GetMeterConsumptionStatsRow, error) {
	return fake.stats, fake.statsErr
}

func (fake *fakeRepository) EarliestReadingTime(ctx context.Context, id uuid.UUID) (time.Time, bool, error) {
	return fake.earliest, fake.hasReadings, nil
}

func (fake *fakeRepository) HourlyProfile(ctx context.Context, params db.GetMeterHourlyProfileParams) ([]db.GetMeterHourlyProfileRow, error) {
	fake.profileParams = params

	return fake.profile, nil
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
			service := NewService(repository, bogota)

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
			service := NewService(repository, bogota)

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
	service := NewService(repository, bogota)

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
			service := NewService(&fakeRepository{err: tableTest.repositoryErr}, bogota)

			_, getErr := service.Get(context.Background(), uuid.New())
			assertStatus(t, getErr, tableTest.wantStatus)

			deleteErr := service.Delete(context.Background(), uuid.New())
			assertStatus(t, deleteErr, tableTest.wantStatus)
		})
	}
}

func TestServiceSeriesRange(t *testing.T) {
	day := 24 * time.Hour
	// Última lectura del dataset: 14-sep 23:00 en Bogotá = 15-sep 04:00 UTC
	latest := time.Date(2026, 9, 15, 4, 0, 0, 0, time.UTC)
	anchor := time.Date(2026, 9, 12, 5, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		query       SeriesQuery
		hasReadings bool
		meterErr    error
		wantStatus  int
		wantFrom    time.Time
		wantTo      time.Time
	}{
		{
			name:        "Default ends at the hour after the latest reading",
			query:       SeriesQuery{},
			hasReadings: true,
			wantFrom:    time.Date(2026, 9, 8, 5, 0, 0, 0, time.UTC),
			wantTo:      time.Date(2026, 9, 15, 5, 0, 0, 0, time.UTC),
		},
		{
			name:     "Only from moves 7 days forward",
			query:    SeriesQuery{From: &anchor},
			wantFrom: anchor,
			wantTo:   anchor.Add(7 * day),
		},
		{
			name:     "Only to moves 7 days back",
			query:    SeriesQuery{To: &anchor},
			wantFrom: anchor.Add(-7 * day),
			wantTo:   anchor,
		},
		{
			name:     "Both dates are used as sent",
			query:    SeriesQuery{From: timePtr(anchor.Add(-2 * day)), To: &anchor},
			wantFrom: anchor.Add(-2 * day),
			wantTo:   anchor,
		},
		{name: "From after to is rejected", query: SeriesQuery{From: &anchor, To: timePtr(anchor.Add(-day))}, wantStatus: http.StatusBadRequest},
		{name: "Range above 31 days is rejected", query: SeriesQuery{From: timePtr(anchor.Add(-40 * day)), To: &anchor}, wantStatus: http.StatusBadRequest},
		{name: "Missing meter is 404", query: SeriesQuery{}, meterErr: ErrNotFound, wantStatus: http.StatusNotFound},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{
				err:           tableTest.meterErr,
				latestReading: latest,
				hasReadings:   tableTest.hasReadings,
			}
			service := NewService(repository, bogota)

			response, err := service.ListReadings(context.Background(), uuid.New(), tableTest.query)

			assertStatus(t, err, tableTest.wantStatus)

			if tableTest.wantStatus != 0 {
				return
			}

			params := repository.readingsParams

			if !params.FromTime.Equal(tableTest.wantFrom) || !params.ToTime.Equal(tableTest.wantTo) {
				t.Fatalf("range sent to repository = [%s, %s), want [%s, %s)",
					params.FromTime.UTC(), params.ToTime.UTC(), tableTest.wantFrom, tableTest.wantTo)
			}

			if !response.From.Equal(tableTest.wantFrom) || !response.To.Equal(tableTest.wantTo) {
				t.Fatalf("range in response = [%s, %s), want [%s, %s)", response.From, response.To, tableTest.wantFrom, tableTest.wantTo)
			}

			if response.Data == nil {
				t.Fatal("data must be an empty slice, not nil (JSON [] instead of null)")
			}
		})
	}
}

func TestServiceSeriesWithoutReadings(t *testing.T) {
	repository := &fakeRepository{hasReadings: false}
	service := NewService(repository, bogota)

	before := time.Now()
	response, err := service.ListReadings(context.Background(), uuid.New(), SeriesQuery{})

	assertStatus(t, err, 0)

	// Sin lecturas, la ventana termina en la hora siguiente a "ahora" y dura 7 días
	if response.To.Before(before) || response.To.Sub(response.From) != 7*24*time.Hour {
		t.Fatalf("range = [%s, %s), want 7 days ending after now", response.From, response.To)
	}
}

func TestServiceListEvents(t *testing.T) {
	eventTime := time.Date(2026, 9, 12, 19, 0, 0, 0, time.UTC)

	repository := &fakeRepository{
		hasReadings:   true,
		latestReading: time.Date(2026, 9, 15, 4, 0, 0, 0, time.UTC),
		events: []db.ListEventsByMeterRow{
			{UidEvent: uuid.New(), DtmTimestampEvent: eventTime, StrTypeEvent: db.EventTypeUNKNOWN, StrDescriptionEvent: "No operational event reported"},
		},
	}
	service := NewService(repository, bogota)

	response, err := service.ListEvents(context.Background(), uuid.New(), SeriesQuery{})

	assertStatus(t, err, 0)

	// Mismo rango por defecto que las lecturas: los marcadores caen dentro de la gráfica
	if !repository.eventsParams.ToTime.Equal(time.Date(2026, 9, 15, 5, 0, 0, 0, time.UTC)) {
		t.Fatalf("events range to = %s, want same default as readings", repository.eventsParams.ToTime)
	}

	if len(response.Data) != 1 || response.Data[0].Type != db.EventTypeUNKNOWN || !response.Data[0].Timestamp.Equal(eventTime) {
		t.Fatalf("unexpected events: %+v", response.Data)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func TestServiceListAttachesStats(t *testing.T) {
	meterID := uuid.New()
	lastReading := time.Date(2026, 9, 15, 4, 0, 0, 0, time.UTC)

	statsOf := func(id uuid.UUID) db.GetMeterConsumptionStatsRow {
		return db.GetMeterConsumptionStatsRow{
			UidMeter:       id,
			BaselineAvgKwh: 43.7,
			RecentAvgKwh:   91.69,
			VariationPct:   109.8,
			MinPowerFactor: 0.708,
			TotalKwh:       17526.04,
			LastReadingAt:  lastReading,
		}
	}

	tests := []struct {
		name       string
		stats      []db.GetMeterConsumptionStatsRow
		statsErr   error
		wantStatus int
		wantStats  bool
	}{
		// Si attachStats recorriera con `for _, response := range` (copias), Stats quedaría en nil y este caso fallaría
		{name: "Meter with readings gets its stats", stats: []db.GetMeterConsumptionStatsRow{statsOf(meterID)}, wantStats: true},
		{name: "Stats of other meters are not mixed", stats: []db.GetMeterConsumptionStatsRow{statsOf(uuid.New())}, wantStats: false},
		{name: "Meter without readings has nil stats", stats: nil, wantStats: false},
		{name: "Stats query failure is 500", statsErr: errors.New("connection refused"), wantStatus: http.StatusInternalServerError},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{
				meter:    db.Meter{UidMeter: meterID, StrCodeMeter: "M-109", StrStatusMeter: db.MeterStatusOK},
				stats:    tableTest.stats,
				statsErr: tableTest.statsErr,
			}
			service := NewService(repository, bogota)

			page, err := service.List(context.Background(), ListQuery{})

			assertStatus(t, err, tableTest.wantStatus)

			if tableTest.wantStatus != 0 {
				return
			}

			stats := page.Data[0].Stats

			if !tableTest.wantStats {
				if stats != nil {
					t.Fatalf("stats = %+v, want nil", stats)
				}
				return
			}

			if stats == nil {
				t.Fatal("stats = nil, want the meter's stats attached")
			}

			want := StatsResponse{
				BaselineKwh:    43.7,
				RecentKwh:      91.69,
				VariationPct:   109.8,
				MinPowerFactor: 0.708,
				TotalKwh:       17526.04,
				LastReadingAt:  lastReading,
			}

			if *stats != want {
				t.Fatalf("stats = %+v, want %+v", *stats, want)
			}
		})
	}
}

// bogota es la zona horaria del sitio en los tests (la misma de APP_TIMEZONE).
// Se carga UNA vez al iniciar el paquete de tests; MustLoad no existe, así que un helper hace el panic.
var bogota = mustLoadLocation("America/Bogota")

func mustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}

	return location
}

func TestServiceHourlyProfile(t *testing.T) {
	day := 24 * time.Hour
	// Primera lectura del dataset: 1-sep 00:00 en Bogotá = 1-sep 05:00 UTC
	earliest := time.Date(2026, 9, 1, 5, 0, 0, 0, time.UTC)
	anchor := time.Date(2026, 9, 12, 5, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		query       SeriesQuery
		hasReadings bool
		wantFrom    time.Time
		wantTo      time.Time
	}{
		{
			name:        "Default is the first week of data",
			query:       SeriesQuery{},
			hasReadings: true,
			wantFrom:    earliest,
			wantTo:      earliest.Add(7 * day),
		},
		{
			name:     "Explicit range overrides the default",
			query:    SeriesQuery{From: timePtr(anchor.Add(-2 * day)), To: &anchor},
			wantFrom: anchor.Add(-2 * day),
			wantTo:   anchor,
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{
				earliest:    earliest,
				hasReadings: tableTest.hasReadings,
				profile: []db.GetMeterHourlyProfileRow{
					{HourOfDay: 14, AvgKwh: 51.31, ReadingsCount: 7},
				},
			}
			service := NewService(repository, bogota)

			response, err := service.HourlyProfile(context.Background(), uuid.New(), tableTest.query)

			assertStatus(t, err, 0)

			params := repository.profileParams

			if !params.FromTime.Equal(tableTest.wantFrom) || !params.ToTime.Equal(tableTest.wantTo) {
				t.Fatalf("range = [%s, %s), want [%s, %s)", params.FromTime.UTC(), params.ToTime.UTC(), tableTest.wantFrom, tableTest.wantTo)
			}

			// La zona viaja a Postgres (AT TIME ZONE) y vuelve en la respuesta
			if params.Timezone != "America/Bogota" || response.Timezone != "America/Bogota" {
				t.Fatalf("timezone sent/returned = %q/%q, want America/Bogota", params.Timezone, response.Timezone)
			}

			want := HourlyValue{Hour: 14, AvgKwh: 51.31, Samples: 7}
			if len(response.Hours) != 1 || response.Hours[0] != want {
				t.Fatalf("hours = %+v, want [%+v]", response.Hours, want)
			}
		})
	}
}

func TestServiceReadingsAndProfileUseOppositeAnchors(t *testing.T) {
	repository := &fakeRepository{
		hasReadings:   true,
		earliest:      time.Date(2026, 9, 1, 5, 0, 0, 0, time.UTC),
		latestReading: time.Date(2026, 9, 15, 4, 0, 0, 0, time.UTC),
	}
	service := NewService(repository, bogota)

	readings, err := service.ListReadings(context.Background(), uuid.New(), SeriesQuery{})
	assertStatus(t, err, 0)

	profile, err := service.HourlyProfile(context.Background(), uuid.New(), SeriesQuery{})
	assertStatus(t, err, 0)

	// Gráfica: la ÚLTIMA semana (8 al 14 de sep). Curva normal: la PRIMERA (1 al 7 de sep).
	if !readings.From.Equal(time.Date(2026, 9, 8, 5, 0, 0, 0, time.UTC)) {
		t.Fatalf("readings from = %s, want last week", readings.From)
	}

	if !profile.From.Equal(time.Date(2026, 9, 1, 5, 0, 0, 0, time.UTC)) {
		t.Fatalf("profile from = %s, want first week", profile.From)
	}
}
