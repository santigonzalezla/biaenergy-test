-- name: InsertAnomaly :exec
INSERT INTO anomaly (uid_analysis, uid_meter, uid_event, str_type_anomaly, str_severity_anomaly, str_rule_id_anomaly,
                     dec_confidence_anomaly, dec_priority_score_anomaly, dec_baseline_kwh_anomaly,
                     dec_current_kwh_anomaly, dec_variation_pct_anomaly, dtm_window_start_anomaly,
                     dtm_window_end_anomaly, str_reason_anomaly, str_recommended_action_anomaly,
                     arr_changed_variables_anomaly, json_evidence_anomaly, dtm_detected_at_anomaly)
VALUES (sqlc.arg('analysis_id'), sqlc.arg('meter_id'), sqlc.narg('event_id'), sqlc.arg('type'), sqlc.arg('severity'),
        sqlc.arg('rule_id'), sqlc.arg('confidence'), sqlc.arg('priority_score'), sqlc.arg('baseline_kwh'),
        sqlc.arg('current_kwh'), sqlc.arg('variation_pct'), sqlc.arg('window_start'), sqlc.narg('window_end'),
        sqlc.arg('reason'), sqlc.arg('recommended_action'), sqlc.arg('changed_variables'), sqlc.arg('evidence'),
        sqlc.arg('detected_at'));

-- name: ListAnomalies :many
-- Anomalías de un análisis, de la más a la menos prioritaria, con el código y nombre del medidor para la tabla.
SELECT a.uid_anomaly,
       a.num_id_anomaly,
       a.uid_meter,
       m.str_code_meter,
       m.str_name_meter,
       a.str_type_anomaly,
       a.str_severity_anomaly,
       a.str_status_anomaly,
       a.str_rule_id_anomaly,
       a.dec_confidence_anomaly,
       a.dec_priority_score_anomaly,
       a.dec_variation_pct_anomaly,
       a.dtm_detected_at_anomaly,
       a.str_reason_anomaly
FROM anomaly a
         JOIN meter m ON m.uid_meter = a.uid_meter
WHERE a.uid_analysis = sqlc.arg('analysis_id')
  AND (sqlc.narg('type')::anomaly_type IS NULL OR a.str_type_anomaly = sqlc.narg('type'))
  AND (sqlc.narg('severity')::anomaly_severity IS NULL OR a.str_severity_anomaly = sqlc.narg('severity'))
  AND (sqlc.narg('status')::anomaly_status IS NULL OR a.str_status_anomaly = sqlc.narg('status'))
ORDER BY a.dec_priority_score_anomaly DESC, a.num_id_anomaly;

-- name: GetAnomaly :one
-- Detalle completo para la pantalla de investigación, con el medidor y el evento relacionado.
SELECT a.*,
       m.str_code_meter,
       m.str_name_meter,
       m.str_location_meter,
       e.str_type_event,
       e.dtm_timestamp_event,
       e.str_description_event
FROM anomaly a
         JOIN meter m ON m.uid_meter = a.uid_meter
         LEFT JOIN event e ON e.uid_event = a.uid_event
WHERE a.uid_anomaly = $1;

-- name: UpdateAnomalyStatus :one
-- La acción del operador. La nota de resolución solo se reemplaza si llega una nueva.
UPDATE anomaly
SET str_status_anomaly          = sqlc.arg('status'),
    str_resolution_note_anomaly = COALESCE(sqlc.narg('resolution_note'), str_resolution_note_anomaly)
WHERE uid_anomaly = sqlc.arg('id')
RETURNING *;
