package ingestion

import (
	"strings"
	"testing"
	"time"
)

const readingsHeader = "meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status\n"

func bogota(t *testing.T) *time.Location {
	t.Helper()

	location, err := time.LoadLocation("America/Bogota")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	return location
}

func TestParseReadings(t *testing.T) {
	tests := []struct {
		name         string
		csv          string
		wantRows     int
		wantFileErr  bool
		wantRowErrs  int
		wantErrorCol string
	}{
		{name: "Valid rows", csv: readingsHeader + "M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\nm-102,2026-09-01 01:00:00,20.1,220,99,0.93,\n", wantRows: 2},
		{name: "Columns in another order", csv: "status,meter_id,timestamp,power_factor,current_a,voltage_v,consumption_kwh\nOK,M-101,2026-09-01 00:00:00,0.9,100,220,20\n", wantRows: 1},
		{name: "Header with BOM from Excel", csv: "\ufeff" + readingsHeader + "M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n", wantRows: 1},
		{name: "Empty file", csv: "", wantFileErr: true},
		{name: "Missing column", csv: "meter_id,timestamp,consumption_kwh\nM-101,2026-09-01 00:00:00,1\n", wantFileErr: true},
		{name: "Not a number", csv: readingsHeader + "M-101,2026-09-01 00:00:00,abc,221.9,101.28,0.954,OK\n", wantRowErrs: 1, wantErrorCol: "consumption_kwh"},
		{name: "Power factor above 1", csv: readingsHeader + "M-101,2026-09-01 00:00:00,23.5,221.9,101.28,1.4,OK\n", wantRowErrs: 1, wantErrorCol: "power_factor"},
		{name: "Negative consumption", csv: readingsHeader + "M-101,2026-09-01 00:00:00,-3,221.9,101.28,0.9,OK\n", wantRowErrs: 1, wantErrorCol: "consumption_kwh"},
		{name: "Invalid date", csv: readingsHeader + "M-101,01/09/2026,23.5,221.9,101.28,0.954,OK\n", wantRowErrs: 1, wantErrorCol: "timestamp"},
		{name: "Invalid meter code", csv: readingsHeader + "M 101!,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n", wantRowErrs: 1, wantErrorCol: "meter_id"},
		{name: "Wrong number of fields", csv: readingsHeader + "M-101,2026-09-01 00:00:00,23.5\n", wantRowErrs: 1},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			rows, rowErrs, err := ParseReadings(strings.NewReader(tableTest.csv), bogota(t))

			if (err != nil) != tableTest.wantFileErr {
				t.Fatalf("file error = %v, want error: %v", err, tableTest.wantFileErr)
			}

			if len(rows) != tableTest.wantRows {
				t.Fatalf("rows = %d, want %d", len(rows), tableTest.wantRows)
			}

			if len(rowErrs) != tableTest.wantRowErrs {
				t.Fatalf("row errors = %d, want %d (%+v)", len(rowErrs), tableTest.wantRowErrs, rowErrs)
			}

			if tableTest.wantErrorCol != "" {
				if rowErrs[0].Column != tableTest.wantErrorCol || rowErrs[0].Line != 2 {
					t.Fatalf("error = %+v, want column %q on line 2", rowErrs[0], tableTest.wantErrorCol)
				}
			}
		})
	}
}

func TestParseReadingsNormalization(t *testing.T) {
	csv := readingsHeader + " m-109 ,2026-09-12 14:00:00,92.8,213,507,0.71,\n"

	rows, rowErrs, err := ParseReadings(strings.NewReader(csv), bogota(t))
	if err != nil || len(rowErrs) > 0 {
		t.Fatalf("unexpected errors: %v %+v", err, rowErrs)
	}

	row := rows[0]

	if row.MeterCode != "M-109" || row.Status != "OK" {
		t.Fatalf("code/status = %q/%q, want M-109/OK", row.MeterCode, row.Status)
	}

	// 14:00 en Bogotá (UTC-5) es 19:00 UTC: la hora del CSV se interpreta en la zona del sitio
	want := time.Date(2026, 9, 12, 19, 0, 0, 0, time.UTC)
	if !row.Timestamp.Equal(want) {
		t.Fatalf("timestamp = %s, want %s", row.Timestamp.UTC(), want)
	}
}

func TestParseReadingsStopsAtMaxErrors(t *testing.T) {
	csv := readingsHeader + strings.Repeat("M-101,bad-date,1,220,10,0.9,OK\n", 50)

	_, rowErrs, err := ParseReadings(strings.NewReader(csv), bogota(t))
	if err != nil {
		t.Fatalf("unexpected file error: %v", err)
	}

	if len(rowErrs) != maxRowErrors {
		t.Fatalf("row errors = %d, want %d", len(rowErrs), maxRowErrors)
	}
}

func TestParseEvents(t *testing.T) {
	header := "meter_id,event_timestamp,event_type,description\n"

	tests := []struct {
		name        string
		csv         string
		wantRows    int
		wantRowErrs int
	}{
		{name: "Dataset format without seconds", csv: header + "M-104,2026-09-11 00:00,OPERATIONAL_CHANGE,New production line activated\n", wantRows: 1},
		{name: "Lowercase type", csv: header + "M-106,2026-09-08 00:00,scheduled_outage,Maintenance\n", wantRows: 1},
		{name: "Unknown type", csv: header + "M-109,2026-09-12 14:00,EARTHQUAKE,?\n", wantRowErrs: 1},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			rows, rowErrs, err := ParseEvents(strings.NewReader(tableTest.csv), bogota(t))
			if err != nil {
				t.Fatalf("unexpected file error: %v", err)
			}

			if len(rows) != tableTest.wantRows || len(rowErrs) != tableTest.wantRowErrs {
				t.Fatalf("rows/errors = %d/%d, want %d/%d (%+v)", len(rows), len(rowErrs), tableTest.wantRows, tableTest.wantRowErrs, rowErrs)
			}
		})
	}
}
