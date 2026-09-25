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
