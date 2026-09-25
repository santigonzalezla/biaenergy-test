package ingestion

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/meter"
)

const maxRowErrors = 20

var (
	readingColumns   = []string{"meter_id", "timestamp", "consumption_kwh", "voltage_v", "current_a", "power_factor", "status"}
	eventColumns     = []string{"meter_id", "event_timestamp", "event_type", "description"}
	timestampLayouts = []string{"2006-01-02 15:04:05", "2006-01-02 15:04", time.RFC3339}
)

type ReadingRow struct {
	MeterCode      string
	Timestamp      time.Time
	ConsumptionKwh float64
	Voltage        float64
	Current        float64
	PowerFactor    float64
	Status         string
}

type EventRow struct {
	MeterCode   string
	Timestamp   time.Time
	Type        string
	Description string
}

type RowError struct {
	Line    int    `json:"line"`
	Column  string `json:"colum.omitempty"`
	Message string `json:"message"`
}

func ParseReadings(source io.Reader, location *time.Location) ([]ReadingRow, []RowError, error) {
	var rows []ReadingRow

	rowErrs, err := readRecords(source, readingColumns, func(fields map[string]string) *RowError {
		row, rowErr := parseReading(fields, location)

		if rowErr != nil {
			return rowErr
		}

		rows = append(rows, row)

		return nil
	})

	return rows, rowErrs, err
}

func ParseEvents(source io.Reader, location *time.Location) ([]EventRow, []RowError, error) {
	var rows []EventRow

	rowErrs, err := readRecords(source, eventColumns, func(fields map[string]string) *RowError {
		row, rowErr := parseEvent(fields, location)

		if rowErr != nil {
			return rowErr
		}

		rows = append(rows, row)

		return nil
	})

	return rows, rowErrs, err
}

func parseReading(fields map[string]string, location *time.Location) (ReadingRow, *RowError) {
	code, rowErr := parseMeterCode(fields["meter_id"])

	if rowErr != nil {
		return ReadingRow{}, rowErr
	}

	timestamp, rowErr := parseTimestamp(fields, "timestamp", location)

	if rowErr != nil {
		return ReadingRow{}, rowErr
	}

	consumption, rowErr := parseNumber(fields, "consumption_kwh", 0, math.Inf(1))
	if rowErr != nil {
		return ReadingRow{}, rowErr
	}

	voltage, rowErr := parseNumber(fields, "voltage_v", 0, math.Inf(1))
	if rowErr != nil {
		return ReadingRow{}, rowErr
	}

	current, rowErr := parseNumber(fields, "current_a", 0, math.Inf(1))
	if rowErr != nil {
		return ReadingRow{}, rowErr
	}

	powerFactor, rowErr := parseNumber(fields, "power_factor", 0, 1)
	if rowErr != nil {
		return ReadingRow{}, rowErr
	}

	status := strings.ToUpper(fields["status"])
	if status == "" {
		status = "OK"
	}

	if len(status) > 20 {
		return ReadingRow{}, &RowError{Column: "status", Message: "must be at most 20 characters"}
	}

	return ReadingRow{
		MeterCode:      code,
		Timestamp:      timestamp,
		ConsumptionKwh: consumption,
		Voltage:        voltage,
		Current:        current,
		PowerFactor:    powerFactor,
		Status:         status,
	}, nil
}

func readRecords(source io.Reader, required []string, handleRow func(fields map[string]string) *RowError) ([]RowError, error) {
	reader := csv.NewReader(source)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()

	if errors.Is(err, io.EOF) {
		return nil, errors.New("the file is empty")
	}

	if err != nil {
		return nil, fmt.Errorf("invalid csv header: %w", err)
	}

	index := make(map[string]int, len(header))

	for position, name := range header {
		index[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "\ufeff")))] = position
	}

	var missing []string

	for _, name := range required {
		if _, ok := index[name]; !ok {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required columns: %s", strings.Join(missing, ", "))
	}

	var rowErrs []RowError

	for line := 2; len(rowErrs) < maxRowErrors; line++ {
		record, err := reader.Read()

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			rowErrs = append(rowErrs, RowError{Line: line, Message: err.Error()})
			continue
		}

		fields := make(map[string]string, len(required))
		for _, name := range required {
			fields[name] = strings.TrimSpace(record[index[name]])
		}

		if rowErr := handleRow(fields); rowErr != nil {
			rowErr.Line = line
			rowErrs = append(rowErrs, *rowErr)
		}
	}

	return rowErrs, nil
}

func parseEvent(fields map[string]string, location *time.Location) (EventRow, *RowError) {
	code, rowErr := parseMeterCode(fields["meter_id"])
	if rowErr != nil {
		return EventRow{}, rowErr
	}

	timestamp, rowErr := parseTimestamp(fields, "event_timestamp", location)
	if rowErr != nil {
		return EventRow{}, rowErr
	}

	eventType := db.EventType(strings.ToUpper(fields["event_type"]))
	if !eventType.Valid() {

		return EventRow{}, &RowError{Column: "event_type", Message: fmt.Sprintf("unknown event type %q",
			fields["event_type"])}
	}

	return EventRow{
		MeterCode:   code,
		Timestamp:   timestamp,
		Type:        string(eventType),
		Description: fields["description"],
	}, nil
}

func parseMeterCode(raw string) (string, *RowError) {
	code := strings.ToUpper(raw)

	if !meter.IsValidCode(code) {
		return "", &RowError{Column: "meter_id", Message: fmt.Sprintf("invalid meter code %q", raw)}
	}

	return code, nil
}

func parseTimestamp(fields map[string]string, column string, location *time.Location) (time.Time, *RowError) {
	for _, layout := range timestampLayouts {
		if timestamp, err := time.ParseInLocation(layout, fields[column], location); err == nil {
			return timestamp, nil
		}
	}

	return time.Time{}, &RowError{Column: column, Message: "Must be a date like 2026-09-01 14:00:00"}
}

func parseNumber(fields map[string]string, column string, min, max float64) (float64, *RowError) {
	value, err := strconv.ParseFloat(fields[column], 64)

	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, &RowError{Column: column, Message: "Must be a number"}
	}

	if value < min || value > max {
		return 0, &RowError{Column: column, Message: fmt.Sprintf("Must be between %g and %g", min, max)}
	}

	return value, nil
}
