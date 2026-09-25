-- +goose Up

-- ─── Estadísticas de consumo por medidor ───
-- Para cada medidor con lecturas en [p_from, p_to) devuelve una fila con:
--   baseline_avg_kwh: consumo horario promedio de los PRIMEROS p_baseline_days días de datos del medidor
--                     → cómo consume "normalmente"
--   recent_avg_kwh:   consumo horario promedio de las ÚLTIMAS p_recent_hours horas de datos del medidor
--                     → cómo consume "ahora"
--   variation_pct:    desviación de recent respecto de baseline, en %
-- Las ventanas se miden desde la primera/última lectura DEL MEDIDOR (no desde p_from/p_to),
-- así nunca quedan vacías aunque el rango pedido empiece antes o termine después de los datos.
-- Ninguna columna devuelve NULL (sqlc las mapea a tipos Go no nulos).
-- STABLE: solo lee; con los mismos parámetros devuelve lo mismo dentro de una consulta (Postgres puede optimizar).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION fn_meter_consumption_stats(
    p_from TIMESTAMPTZ,
    p_to TIMESTAMPTZ,
    p_baseline_days INTEGER DEFAULT 7,
    p_recent_hours INTEGER DEFAULT 48
)
    RETURNS TABLE
            (
                uid_meter        UUID,
                readings_count   BIGINT,
                total_kwh        DOUBLE PRECISION,
                baseline_avg_kwh DOUBLE PRECISION,
                recent_avg_kwh   DOUBLE PRECISION,
                variation_pct    DOUBLE PRECISION,
                avg_voltage      DOUBLE PRECISION,
                min_power_factor DOUBLE PRECISION,
                first_reading_at TIMESTAMPTZ,
                last_reading_at  TIMESTAMPTZ
            )
    LANGUAGE sql
    STABLE
AS
$$
WITH in_range AS (SELECT r.*
                  FROM reading r
                  WHERE r.dtm_timestamp_reading >= p_from
                    AND r.dtm_timestamp_reading < p_to),
     bounds AS (SELECT ir.uid_meter,
                       min(ir.dtm_timestamp_reading) AS first_at,
                       max(ir.dtm_timestamp_reading) AS last_at
                FROM in_range ir
                GROUP BY ir.uid_meter),
     stats AS (SELECT ir.uid_meter,
                      count(*)                            AS readings_count,
                      sum(ir.dec_consumption_kwh_reading) AS total_kwh,
                      avg(ir.dec_consumption_kwh_reading)
                      FILTER (WHERE ir.dtm_timestamp_reading < b.first_at + make_interval(days => p_baseline_days))
                                                          AS baseline_avg_kwh,
                      avg(ir.dec_consumption_kwh_reading)
                      FILTER (WHERE ir.dtm_timestamp_reading > b.last_at - make_interval(hours => p_recent_hours))
                                                          AS recent_avg_kwh,
                      avg(ir.dec_voltage_reading)         AS avg_voltage,
                      min(ir.dec_power_factor_reading)    AS min_power_factor,
                      b.first_at,
                      b.last_at
               FROM in_range ir
                        JOIN bounds b ON b.uid_meter = ir.uid_meter
               GROUP BY ir.uid_meter, b.first_at, b.last_at)
SELECT s.uid_meter,
       s.readings_count,
       round(s.total_kwh::numeric, 2)::double precision,
       round(s.baseline_avg_kwh::numeric, 2)::double precision,
       round(s.recent_avg_kwh::numeric, 2)::double precision,
       -- NULLIF evita dividir por cero si el baseline fue 0; en ese caso la variación se reporta como 0
       coalesce(round(((s.recent_avg_kwh / NULLIF(s.baseline_avg_kwh, 0) - 1) * 100)::numeric, 1), 0)::double precision,
       round(s.avg_voltage::numeric, 1)::double precision,
       s.min_power_factor,
       s.first_at,
       s.last_at
FROM stats s;
$$;
-- +goose StatementEnd

-- ─── Perfil horario de un medidor ───
-- Consumo promedio por hora del día (0-23) en la zona horaria del sitio.
-- Es la curva "normal" que la gráfica de detalle dibuja detrás del consumo real.
-- AT TIME ZONE convierte el instante a hora local: sin eso, "hora 14" sería 14:00 UTC (= 09:00 en Bogotá).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION fn_meter_hourly_profile(
    p_meter_id UUID,
    p_from TIMESTAMPTZ,
    p_to TIMESTAMPTZ,
    p_timezone TEXT
)
    RETURNS TABLE
            (
                hour_of_day    INTEGER,
                avg_kwh        DOUBLE PRECISION,
                readings_count BIGINT
            )
    LANGUAGE sql
    STABLE
AS
$$
SELECT extract(HOUR FROM r.dtm_timestamp_reading AT TIME ZONE p_timezone)::integer,
       round(avg(r.dec_consumption_kwh_reading)::numeric, 2)::double precision,
       count(*)
FROM reading r
WHERE r.uid_meter = p_meter_id
  AND r.dtm_timestamp_reading >= p_from
  AND r.dtm_timestamp_reading < p_to
GROUP BY 1
ORDER BY 1;
$$;
-- +goose StatementEnd

-- ─── Resumen del dashboard ───
-- Una sola fila con los contadores globales. Las columnas de anomalías valen 0
-- hasta que exista el primer análisis (Fase 7).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION fn_dashboard_summary()
    RETURNS TABLE
            (
                meters_count            BIGINT,
                meters_with_alerts      BIGINT,
                readings_count          BIGINT,
                total_kwh               DOUBLE PRECISION,
                open_anomalies          BIGINT,
                high_priority_anomalies BIGINT
            )
    LANGUAGE sql
    STABLE
AS
$$
SELECT (SELECT count(*) FROM meter WHERE dtm_deleted_at_meter IS NULL),
       (SELECT count(*) FROM meter WHERE dtm_deleted_at_meter IS NULL AND str_status_meter <> 'OK'),
       (SELECT count(*) FROM reading),
       (SELECT round(coalesce(sum(dec_consumption_kwh_reading), 0)::numeric, 2)::double precision FROM reading),
       (SELECT count(*) FROM anomaly WHERE str_status_anomaly IN ('OPEN', 'INVESTIGATING')),
       (SELECT count(*)
        FROM anomaly
        WHERE str_status_anomaly IN ('OPEN', 'INVESTIGATING')
          AND str_severity_anomaly = 'HIGH');
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS fn_dashboard_summary();
DROP FUNCTION IF EXISTS fn_meter_hourly_profile(UUID, TIMESTAMPTZ, TIMESTAMPTZ, TEXT);
DROP FUNCTION IF EXISTS fn_meter_consumption_stats(TIMESTAMPTZ, TIMESTAMPTZ, INTEGER, INTEGER);
