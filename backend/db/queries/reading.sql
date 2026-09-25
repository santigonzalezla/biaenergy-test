-- name: InsertReadings :execrows
-- Inserción masiva: cada parámetro es un array (una "columna") y unnest los convierte en filas.
-- ON CONFLICT hace la importación idempotente: re-importar el mismo archivo no duplica lecturas.
INSERT INTO reading (uid_meter, dtm_timestamp_reading, dec_consumption_kwh_reading, dec_voltage_reading,
                     dec_current_reading, dec_power_factor_reading, str_status_reading)
SELECT unnest(sqlc.arg('meter_ids')::uuid[]),
       unnest(sqlc.arg('timestamps')::timestamptz[]),
       unnest(sqlc.arg('consumptions')::double precision[]),
       unnest(sqlc.arg('voltages')::double precision[]),
       unnest(sqlc.arg('currents')::double precision[]),
       unnest(sqlc.arg('power_factors')::double precision[]),
       unnest(sqlc.arg('statuses')::text[])
ON CONFLICT (uid_meter, dtm_timestamp_reading) DO NOTHING;

-- name: ListReadingsByMeter :many
-- Lecturas de un medidor en el rango [from, to): incluye from, excluye to.
-- Usa el índice de uq_reading_meter_timestamp (uid_meter, dtm_timestamp_reading) → no recorre la tabla completa.
SELECT dtm_timestamp_reading,
       dec_consumption_kwh_reading,
       dec_voltage_reading,
       dec_current_reading,
       dec_power_factor_reading,
       str_status_reading
FROM reading
WHERE uid_meter = sqlc.arg('meter_id')
  AND dtm_timestamp_reading >= sqlc.arg('from_time')
  AND dtm_timestamp_reading < sqlc.arg('to_time')
ORDER BY dtm_timestamp_reading;

-- name: GetLatestReadingTime :one
-- Fecha de la lectura más reciente de un medidor (para el rango por defecto de la gráfica).
-- ORDER BY ... DESC LIMIT 1 recorre el índice (uid_meter, dtm_timestamp_reading) desde el final: lee UNA sola fila.
-- Si el medidor no tiene lecturas, :one devuelve pgx.ErrNoRows.
SELECT dtm_timestamp_reading
FROM reading
WHERE uid_meter = $1
ORDER BY dtm_timestamp_reading DESC
LIMIT 1;
