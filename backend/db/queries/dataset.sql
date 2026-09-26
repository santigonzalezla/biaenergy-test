-- Dataset que el backend envía al ai-service. meter_ids NULL = toda la flota activa.

-- name: ListMetersForAnalysis :many
SELECT uid_meter,
       str_code_meter,
       dec_nominal_voltage_meter,
       dec_max_current_meter
FROM meter
WHERE dtm_deleted_at_meter IS NULL
  AND (sqlc.narg('meter_ids')::uuid[] IS NULL OR uid_meter = ANY (sqlc.narg('meter_ids')::uuid[]))
ORDER BY str_code_meter;

-- name: ListReadingsForAnalysis :many
SELECT r.uid_meter,
       r.dtm_timestamp_reading,
       r.dec_consumption_kwh_reading,
       r.dec_voltage_reading,
       r.dec_current_reading,
       r.dec_power_factor_reading
FROM reading r
         JOIN meter m ON m.uid_meter = r.uid_meter
WHERE m.dtm_deleted_at_meter IS NULL
  AND (sqlc.narg('meter_ids')::uuid[] IS NULL OR r.uid_meter = ANY (sqlc.narg('meter_ids')::uuid[]))
ORDER BY r.uid_meter, r.dtm_timestamp_reading;

-- name: ListEventsForAnalysis :many
SELECT e.uid_event,
       e.uid_meter,
       e.dtm_timestamp_event,
       e.str_type_event,
       e.str_description_event
FROM event e
         JOIN meter m ON m.uid_meter = e.uid_meter
WHERE m.dtm_deleted_at_meter IS NULL
  AND (sqlc.narg('meter_ids')::uuid[] IS NULL OR e.uid_meter = ANY (sqlc.narg('meter_ids')::uuid[]))
ORDER BY e.dtm_timestamp_event;
