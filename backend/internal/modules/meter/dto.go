package meter

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

var codePattern = regexp.MustCompile(`^[A-Z0-9-]{2,20}$`) // [2-20] - [mayus, num, hyph]

// IsValidCode indica si un código de medidor ya normalizado (mayúsculas, sin espacios) cumple el formato
func IsValidCode(code string) bool {
	return codePattern.MatchString(code)
}

type CreateMeterRequest struct {
	Code              string   `json:"code"`
	Name              string   `json:"name"`
	Location          string   `json:"location"`
	Sector            string   `json:"sector"`
	NominalVoltage    *float64 `json:"nominalVoltage"`
	MaxCurrent        *float64 `json:"maxCurrent"`
	ContractedPowerKw *float64 `json:"contractedPowerKw"`
}

func (request CreateMeterRequest) Validate() map[string]string {
	errs := map[string]string{}

	if !codePattern.MatchString(strings.ToUpper(strings.TrimSpace(request.Code))) {
		errs["code"] = "Code must be 2-20 characters long: letters, numbers, and hyphen."
	}

	if name := strings.TrimSpace(request.Name); name == "" || len(name) > 120 {
		errs["name"] = "Name is required and must be 120 characters long at most."
	}

	if request.NominalVoltage != nil && *request.NominalVoltage <= 0 {
		errs["nominalVoltage"] = "Nominal voltage must be a positive number."
	}

	if request.MaxCurrent != nil && *request.MaxCurrent <= 0 {
		errs["maxCurrent"] = "Max current must be a positive number."
	}

	if request.ContractedPowerKw != nil && *request.ContractedPowerKw <= 0 {
		errs["contractedPowerKw"] = "Contracted power must be a positive number."
	}

	return errs
}

type UpdateMeterRequest struct {
	Name              *string  `json:"name"`
	Location          *string  `json:"location"`
	Sector            *string  `json:"sector"`
	NominalVoltage    *float64 `json:"nominalVoltage"`
	MaxCurrent        *float64 `json:"maxCurrent"`
	ContractedPowerKw *float64 `json:"contractedPowerKw"`
	Status            *string  `json:"status"`
}

func (request UpdateMeterRequest) Validate() map[string]string {
	errs := map[string]string{}

	if request.Name != nil {
		if name := strings.TrimSpace(*request.Name); name == "" || len(name) > 120 {
			errs["name"] = "Name is required and must be 120 characters long at most."
		}
	}

	if request.NominalVoltage != nil && *request.NominalVoltage <= 0 {
		errs["nominalVoltage"] = "Nominal voltage must be a positive number."
	}

	if request.MaxCurrent != nil && *request.MaxCurrent <= 0 {
		errs["maxCurrent"] = "Max current must be a positive number."
	}

	if request.ContractedPowerKw != nil && *request.ContractedPowerKw <= 0 {
		errs["contractedPowerKw"] = "Contracted power must be a positive number."
	}

	if request.Status != nil && !db.MeterStatus(*request.Status).Valid() {
		errs["status"] = "Status must be with in OK, ALERT, CRITICAL"
	}

	return errs
}

type MeterResponse struct {
	ID                uuid.UUID      `json:"id"`
	NumID             int32          `json:"numId"`
	Code              string         `json:"code"`
	Name              string         `json:"name"`
	Location          string         `json:"location"`
	Sector            string         `json:"sector"`
	NominalVoltage    float64        `json:"nominalVoltage"`
	MaxCurrent        *float64       `json:"maxCurrent"`
	ContractedPowerKw *float64       `json:"contractedPowerKw"`
	Status            db.MeterStatus `json:"status"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

func toMeterResponse(meter db.Meter) MeterResponse {
	return MeterResponse{
		ID:                meter.UidMeter,
		NumID:             meter.NumIDMeter,
		Code:              meter.StrCodeMeter,
		Name:              meter.StrNameMeter,
		Location:          meter.StrLocationMeter,
		Sector:            meter.StrSectorMeter,
		NominalVoltage:    meter.DecNominalVoltageMeter,
		MaxCurrent:        meter.DecMaxCurrentMeter,
		ContractedPowerKw: meter.DecContractedPowerKwMeter,
		Status:            meter.StrStatusMeter,
		CreatedAt:         meter.DtmCreatedAt.UTC(),
		UpdatedAt:         meter.DtmUpdatedAt.UTC(),
	}
}

func toMeterResponses(meters []db.Meter) []MeterResponse {
	responses := make([]MeterResponse, 0, len(meters))

	for _, meter := range meters {
		responses = append(responses, toMeterResponse(meter))
	}

	return responses
}
