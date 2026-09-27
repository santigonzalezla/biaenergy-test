-- name: CreateAnalysis :one
INSERT INTO analysis (uid_user)
VALUES (sqlc.narg('user_id'))
RETURNING *;

-- name: GetAnalysis :one
SELECT *
FROM analysis
WHERE uid_analysis = $1;

-- name: GetLatestAnalysis :one
SELECT *
FROM analysis
ORDER BY dtm_created_at DESC
LIMIT 1;

-- name: GetActiveAnalysis :one
-- Un análisis en curso (PENDING o RUNNING): evita lanzar dos a la vez.
SELECT *
FROM analysis
WHERE str_status_analysis IN ('PENDING', 'RUNNING')
ORDER BY dtm_created_at DESC
LIMIT 1;

-- name: AdvanceAnalysis :exec
-- Marca el paso actual del worker (lo que muestra el stepper del frontend).
-- COALESCE conserva la hora de inicio del primer paso.
UPDATE analysis
SET str_status_analysis     = 'RUNNING',
    str_current_step_analysis = sqlc.arg('step')::text,
    num_progress_analysis   = sqlc.arg('progress'),
    dtm_started_at_analysis = COALESCE(dtm_started_at_analysis, now())
WHERE uid_analysis = sqlc.arg('id');

-- name: CompleteAnalysis :exec
UPDATE analysis
SET str_status_analysis        = 'COMPLETED',
    str_current_step_analysis  = 'completed',
    num_progress_analysis      = 100,
    str_provider_analysis      = sqlc.narg('provider'),
    num_meters_analyzed        = sqlc.arg('meters_analyzed'),
    num_anomalies_analysis     = sqlc.arg('anomalies'),
    num_high_priority_analysis = sqlc.arg('high_priority'),
    dtm_finished_at_analysis   = now()
WHERE uid_analysis = sqlc.arg('id');

-- name: FailAnalysis :exec
UPDATE analysis
SET str_status_analysis      = 'FAILED',
    str_error_analysis       = sqlc.arg('error_message'),
    dtm_finished_at_analysis = now()
WHERE uid_analysis = sqlc.arg('id');

-- name: FailInterruptedAnalyses :execrows
-- Al arrancar el backend: un análisis que quedó PENDING o RUNNING fue interrumpido por un reinicio.
UPDATE analysis
SET str_status_analysis      = 'FAILED',
    str_error_analysis       = 'interrupted by a service restart',
    dtm_finished_at_analysis = now()
WHERE str_status_analysis IN ('PENDING', 'RUNNING');

-- name: GetLatestCompletedAnalysisID :one
-- Análisis cuyas anomalías se muestran por defecto: el último que terminó bien (uno en curso aún no tiene resultados).
SELECT uid_analysis
FROM analysis
WHERE str_status_analysis = 'COMPLETED'
ORDER BY dtm_finished_at_analysis DESC
LIMIT 1;

-- name: RefreshMeterStatuses :exec
-- Estado de cada medidor analizado según sus anomalías en este análisis. meter_ids NULL = toda la flota activa.
UPDATE meter m
SET str_status_meter = CASE
                           WHEN EXISTS (SELECT 1
                                        FROM anomaly a
                                        WHERE a.uid_analysis = sqlc.arg('analysis_id')
                                          AND a.uid_meter = m.uid_meter
                                          AND a.str_type_anomaly = 'REAL_ANOMALY'
                                          AND a.str_severity_anomaly = 'HIGH') THEN 'CRITICAL'
                           WHEN EXISTS (SELECT 1
                                        FROM anomaly a
                                        WHERE a.uid_analysis = sqlc.arg('analysis_id')
                                          AND a.uid_meter = m.uid_meter
                                          AND a.str_type_anomaly IN ('REAL_ANOMALY', 'DATA_QUALITY')) THEN 'ALERT'
                           ELSE 'OK'
    END::meter_status
WHERE m.dtm_deleted_at_meter IS NULL
  AND (sqlc.narg('meter_ids')::uuid[] IS NULL OR m.uid_meter = ANY (sqlc.narg('meter_ids')::uuid[]));
