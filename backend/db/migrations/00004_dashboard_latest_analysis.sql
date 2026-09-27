-- +goose Up

-- Cada análisis guarda su propio conjunto de anomalías: contar todas duplicaba las cifras al re-ejecutar
-- el análisis. El resumen cuenta solo las del último análisis completado, el mismo que muestra la lista.
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
WITH latest_analysis AS (SELECT uid_analysis
                         FROM analysis
                         WHERE str_status_analysis = 'COMPLETED'
                         ORDER BY dtm_finished_at_analysis DESC
                         LIMIT 1),
     current_anomalies AS (SELECT a.*
                           FROM anomaly a
                                    JOIN latest_analysis l ON l.uid_analysis = a.uid_analysis
                           WHERE a.str_status_anomaly IN ('OPEN', 'INVESTIGATING'))
SELECT (SELECT count(*) FROM meter WHERE dtm_deleted_at_meter IS NULL),
       (SELECT count(*) FROM meter WHERE dtm_deleted_at_meter IS NULL AND str_status_meter <> 'OK'),
       (SELECT count(*) FROM reading),
       (SELECT round(coalesce(sum(dec_consumption_kwh_reading), 0)::numeric, 2)::double precision FROM reading),
       (SELECT count(*) FROM current_anomalies),
       (SELECT count(*)
        FROM current_anomalies
        WHERE str_severity_anomaly = 'HIGH'
          AND str_type_anomaly <> 'FALSE_POSITIVE');
$$;
-- +goose StatementEnd

-- +goose Down

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
