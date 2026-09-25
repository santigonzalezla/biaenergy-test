-- name: InsertEvents :execrows
-- Mismo patrón que InsertReadings; el tipo llega como text[] y se convierte al enum en SQL
INSERT INTO event (uid_meter, dtm_timestamp_event, str_type_event, str_description_event)
SELECT unnest(sqlc.arg('meter_ids')::uuid[]),
       unnest(sqlc.arg('timestamps')::timestamptz[]),
       unnest(sqlc.arg('types')::text[])::event_type,
       unnest(sqlc.arg('descriptions')::text[])
ON CONFLICT (uid_meter, dtm_timestamp_event, str_type_event) DO NOTHING;

-- name: ListEventsByMeter :many
-- Eventos de un medidor en el rango [from, to), para los marcadores de la gráfica
SELECT uid_event,
       dtm_timestamp_event,
       str_type_event,
       str_description_event
FROM event
WHERE uid_meter = sqlc.arg('meter_id')
  AND dtm_timestamp_event >= sqlc.arg('from_time')
  AND dtm_timestamp_event < sqlc.arg('to_time')
ORDER BY dtm_timestamp_event;
