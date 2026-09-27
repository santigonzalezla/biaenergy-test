package aiclient

import "time"

type AnalyzeRequest struct {
	Timezone string    `json:"timezone"`
	Meters   []Meter   `json:"meters"`
	Readings []Reading `json:"readings"`
	Events   []Event   `json:"events"`
}

type Meter struct {
	Id             string   `json:"id"`
	Code           string   `json:"code"`
	NominalVoltage float64  `json:"nominalVoltage"`
	MaxCurrent     *float64 `json:"maxCurrent,omitempty"`
}

type Reading struct {
	MeterId        string    `json:"meterId"`
	Timestamp      time.Time `json:"timestamp"`
	ConsumptionKwh float64   `json:"consumptionKwh"`
	Voltage        float64   `json:"voltage"`
	Current        float64   `json:"current"`
	PowerFactor    float64   `json:"powerFactor"`
}

type Event struct {
	Id          string    `json:"id"`
	MeterId     string    `json:"meterId"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
}

type AnalyzeResult struct {
	MetersAnalyzed int       `json:"metersAnalyzed"`
	Findings       []Finding `json:"findings"`
}

type Finding struct {
	MeterId           string            `json:"meterId"`
	MeterCode         string            `json:"meterCode"`
	Type              string            `json:"type"`
	Severity          string            `json:"severity"`
	RuleId            string            `json:"ruleId"`
	Confidence        float64           `json:"confidence"`
	PriorityScore     float64           `json:"priorityScore"`
	DetectedAt        time.Time         `json:"detectedAt"`
	WindowStart       time.Time         `json:"windowStart"`
	WindowEnd         *time.Time        `json:"windowEnd"`
	BaselineKwh       float64           `json:"baselineKwh"`
	CurrentKwh        float64           `json:"currentKwh"`
	VariationPct      float64           `json:"variationPct"`
	ChangedVariables  []ChangedVariable `json:"changedVariables"`
	Signals           []Signal          `json:"signals"`
	RelatedEvent      *EventReference   `json:"relatedEvent"`
	Reason            string            `json:"reason"`
	RecommendedAction string            `json:"recommendedAction"`
}

type ChangedVariable struct {
	Name      string  `json:"name"`
	Baseline  float64 `json:"baseline"`
	Observed  float64 `json:"observed"`
	ChangePct float64 `json:"changePct"`
}

type Signal struct {
	Kind          string     `json:"kind"`
	StartedAt     time.Time  `json:"startedAt"`
	EndedAt       *time.Time `json:"endedAt"`
	Magnitude     float64    `json:"magnitude"`
	BaselineValue float64    `json:"baselineValue"`
	ObservedValue float64    `json:"observedValue"`
	Description   string     `json:"description"`
}

type EventReference struct {
	Id          string    `json:"id"`
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}
