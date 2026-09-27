package aiclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const findingsResponse = `{
	"metersAnalyzed": 12,
	"findings": [{
		"meterId": "m-109", "meterCode": "M-109", "type": "REAL_ANOMALY", "severity": "HIGH",
		"ruleId": "R4_UNEXPLAINED_SURGE", "confidence": 0.9, "priorityScore": 90,
		"detectedAt": "2026-09-12T19:00:00Z", "windowStart": "2026-09-12T19:00:00Z", "windowEnd": null,
		"baselineKwh": 44.07, "currentKwh": 92.77, "variationPct": 110.5,
		"changedVariables": [{"name": "power_factor", "baseline": 0.94, "observed": 0.74, "changePct": -21.3}],
		"signals": [{"kind": "CONSUMPTION_SURGE", "startedAt": "2026-09-12T19:00:00Z", "endedAt": null,
			"magnitude": 116.9, "baselineValue": 42.38, "observedValue": 91.92, "description": "surge"}],
		"relatedEvent": {"id": "e-1", "type": "UNKNOWN", "timestamp": "2026-09-12T19:00:00Z",
			"description": "No operational event reported"},
		"reason": "El consumo aumentó.", "recommendedAction": "Revisar los equipos."
	}]
}`

func sampleRequest() AnalyzeRequest {
	at := time.Date(2026, 9, 12, 14, 0, 0, 0, time.FixedZone("COT", -5*3600))

	return AnalyzeRequest{
		Timezone: "America/Bogota",
		Meters:   []Meter{{Id: "m-109", Code: "M-109", NominalVoltage: 220}},
		Readings: []Reading{{MeterId: "m-109", Timestamp: at, ConsumptionKwh: 110.35, Voltage: 218.24, Current: 492.65, PowerFactor: 0.736}},
		Events:   []Event{{Id: "e-1", MeterId: "m-109", Timestamp: at, Type: "UNKNOWN", Description: "No operational event reported"}},
	}
}

func TestAnalyzeDecodesFindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, findingsResponse)
	}))
	defer server.Close()

	result, err := New(server.URL).Analyze(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if result.MetersAnalyzed != 12 || len(result.Findings) != 1 {
		t.Fatalf("result = %+v, want 12 meters and 1 finding", result)
	}

	finding := result.Findings[0]
	if finding.RuleId != "R4_UNEXPLAINED_SURGE" || finding.PriorityScore != 90 {
		t.Fatalf("finding = %+v", finding)
	}
	if finding.WindowEnd != nil {
		t.Fatal("windowEnd must be nil for an active anomaly")
	}
	if finding.RelatedEvent == nil || finding.RelatedEvent.Type != "UNKNOWN" {
		t.Fatalf("relatedEvent = %+v", finding.RelatedEvent)
	}
	if len(finding.ChangedVariables) != 1 || finding.ChangedVariables[0].Name != "power_factor" {
		t.Fatalf("changedVariables = %+v", finding.ChangedVariables)
	}
}

func TestAnalyzeSendsTheContractTheAIServiceExpects(t *testing.T) {
	var received map[string]any
	var method, path, contentType string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method, path, contentType = request.Method, request.URL.Path, request.Header.Get("Content-Type")
		_ = json.NewDecoder(request.Body).Decode(&received)
		_, _ = io.WriteString(writer, `{"metersAnalyzed": 1, "findings": []}`)
	}))
	defer server.Close()

	if _, err := New(server.URL+"/").Analyze(context.Background(), sampleRequest()); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if method != http.MethodPost || path != "/analyze" || contentType != "application/json" {
		t.Fatalf("request = %s %s (%s), want POST /analyze (application/json)", method, path, contentType)
	}

	meter := received["meters"].([]any)[0].(map[string]any)
	if _, sent := meter["maxCurrent"]; sent {
		t.Fatal("maxCurrent must be omitted when the meter has no rated current")
	}

	reading := received["readings"].([]any)[0].(map[string]any)
	for _, key := range []string{"meterId", "timestamp", "consumptionKwh", "voltage", "current", "powerFactor"} {
		if _, ok := reading[key]; !ok {
			t.Fatalf("reading is missing key %q: %v", key, reading)
		}
	}
	if reading["timestamp"] != "2026-09-12T14:00:00-05:00" {
		t.Fatalf("timestamp = %v, want RFC3339 with offset", reading["timestamp"])
	}

	event := received["events"].([]any)[0].(map[string]any)
	for _, key := range []string{"id", "meterId", "timestamp", "type", "description"} {
		if _, ok := event[key]; !ok {
			t.Fatalf("event is missing key %q: %v", key, event)
		}
	}
}

func TestAnalyzeErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "Validation error from the AI service",
			status:     http.StatusBadRequest,
			body:       `{"error": {"code": "VALIDATION_ERROR", "message": "The request contains invalid data"}}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "Internal error from the AI service",
			status:     http.StatusInternalServerError,
			body:       `{"error": {"code": "INTERNAL_ERROR", "message": "Internal server error"}}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
		{
			name:       "Proxy error without JSON",
			status:     http.StatusBadGateway,
			body:       "<html>Bad Gateway</html>",
			wantStatus: http.StatusBadGateway,
			wantCode:   "UNEXPECTED_RESPONSE",
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(tableTest.status)
				_, _ = io.WriteString(writer, tableTest.body)
			}))
			defer server.Close()

			_, err := New(server.URL).Analyze(context.Background(), sampleRequest())

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %v, want *APIError", err)
			}
			if apiErr.StatusCode != tableTest.wantStatus || apiErr.Code != tableTest.wantCode {
				t.Fatalf("APIError = %+v, want %d %s", apiErr, tableTest.wantStatus, tableTest.wantCode)
			}
		})
	}
}

func TestAnalyzeRejectsAMalformedSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(writer, `{"metersAnalyzed": `)
	}))
	defer server.Close()

	_, err := New(server.URL).Analyze(context.Background(), sampleRequest())

	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("error = %v, want a decode error", err)
	}
}

func slowServer(t *testing.T) *httptest.Server {
	t.Helper()

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-release
	}))

	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	return server
}

func TestAnalyzeRespectsTheTimeout(t *testing.T) {
	server := slowServer(t)

	started := time.Now()
	_, err := New(server.URL, WithTimeout(50*time.Millisecond)).Analyze(context.Background(), sampleRequest())

	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Analyze() took %v, the timeout was not applied", elapsed)
	}
}

func TestAnalyzeStopsWhenTheContextIsCancelled(t *testing.T) {
	server := slowServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := New(server.URL).Analyze(ctx, sampleRequest())

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestWithHTTPClientReplacesTheDefaultClient(t *testing.T) {
	custom := &http.Client{Timeout: 7 * time.Second}

	client := New("http://ai-service:8000", WithHTTPClient(custom))

	if client.httpClient != custom {
		t.Fatal("WithHTTPClient did not replace the http client")
	}
}
