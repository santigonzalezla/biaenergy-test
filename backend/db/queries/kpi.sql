-- name: GetMeterConsumptionStats :many
-- sqlc no deduce las columnas de una función RETURNS TABLE: cada columna se nombra y se castea explícitamente.
SELECT stats.uid_meter::uuid                    AS uid_meter,
       stats.readings_count::bigint             AS readings_count,
       stats.total_kwh::double precision        AS total_kwh,
       stats.baseline_avg_kwh::double precision AS baseline_avg_kwh,
       stats.recent_avg_kwh::double precision   AS recent_avg_kwh,
       stats.variation_pct::double precision    AS variation_pct,
       stats.avg_voltage::double precision      AS avg_voltage,
       stats.min_power_factor::double precision AS min_power_factor,
       stats.first_reading_at::timestamptz      AS first_reading_at,
       stats.last_reading_at::timestamptz       AS last_reading_at
FROM fn_meter_consumption_stats(sqlc.arg('from_time'), sqlc.arg('to_time'),
                                sqlc.arg('baseline_days'), sqlc.arg('recent_hours')) AS stats;

-- name: GetMeterHourlyProfile :many
SELECT profile.hour_of_day::integer          AS hour_of_day,
       profile.avg_kwh::double precision     AS avg_kwh,
       profile.readings_count::bigint        AS readings_count
FROM fn_meter_hourly_profile(sqlc.arg('meter_id'), sqlc.arg('from_time'), sqlc.arg('to_time'),
                             sqlc.arg('timezone')) AS profile;

-- name: GetDashboardSummary :one
SELECT summary.meters_count::bigint            AS meters_count,
       summary.meters_with_alerts::bigint      AS meters_with_alerts,
       summary.readings_count::bigint          AS readings_count,
       summary.total_kwh::double precision     AS total_kwh,
       summary.open_anomalies::bigint          AS open_anomalies,
       summary.high_priority_anomalies::bigint AS high_priority_anomalies
FROM fn_dashboard_summary() AS summary;
