-- name: ListMeters :many
SELECT *
FROM meter
WHERE dtm_deleted_at_meter IS NULL
  AND (sqlc.narg('status')::meter_status IS NULL OR str_status_meter = sqlc.narg('status'))
  AND (sqlc.narg('search')::text IS NULL
    OR str_code_meter ILIKE '%' || sqlc.narg('search') || '%'
    OR str_name_meter ILIKE '%' || sqlc.narg('search') || '%'
    OR str_location_meter ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY CASE
             WHEN sqlc.arg('sort_by')::text = 'code' AND sqlc.arg('sort_desc')::boolean = false THEN
                 str_code_meter END,
         CASE
             WHEN sqlc.arg('sort_by')::text = 'code' AND sqlc.arg('sort_desc')::boolean = true THEN
                 str_code_meter END DESC,
         CASE
             WHEN sqlc.arg('sort_by')::text = 'name' AND sqlc.arg('sort_desc')::boolean = false THEN
                 str_name_meter END,
         CASE
             WHEN sqlc.arg('sort_by')::text = 'name' AND sqlc.arg('sort_desc')::boolean = true THEN
                 str_name_meter END DESC,
         CASE
             WHEN sqlc.arg('sort_by')::text = 'status' AND sqlc.arg('sort_desc')::boolean = false THEN
                 str_status_meter END,
         CASE
             WHEN sqlc.arg('sort_by')::text = 'status' AND sqlc.arg('sort_desc')::boolean = true THEN
                 str_status_meter END DESC,
         str_code_meter
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: CountMeters :one
SELECT count(*)
FROM meter
WHERE dtm_deleted_at_meter IS NULL
  AND (sqlc.narg('status')::meter_status IS NULL OR str_status_meter = sqlc.narg('status'))
  AND (sqlc.narg('search')::text IS NULL
    OR str_code_meter ILIKE '%' || sqlc.narg('search') || '%'
    OR str_name_meter ILIKE '%' || sqlc.narg('search') || '%'
    OR str_location_meter ILIKE '%' || sqlc.narg('search') || '%');

-- name: GetMeterByID :one
SELECT *
FROM meter
WHERE uid_meter = $1
  AND dtm_deleted_at_meter IS NULL;

-- name: CreateMeter :one
INSERT INTO meter (str_code_meter, str_name_meter, str_location_meter, str_sector_meter,
                   dec_nominal_voltage_meter, dec_max_current_meter, dec_contracted_power_kw_meter)
VALUES (sqlc.arg('code'), sqlc.arg('name'), sqlc.arg('location'), sqlc.arg('sector'),
        sqlc.arg('nominal_voltage'), sqlc.narg('max_current'), sqlc.narg('contracted_power_kw'))
RETURNING *;

-- name: UpdateMeter :one
UPDATE meter
SET str_name_meter                = COALESCE(sqlc.narg('name'), str_name_meter),
    str_location_meter            = COALESCE(sqlc.narg('location'), str_location_meter),
    str_sector_meter              = COALESCE(sqlc.narg('sector'), str_sector_meter),
    dec_nominal_voltage_meter     = COALESCE(sqlc.narg('nominal_voltage'),
                                             dec_nominal_voltage_meter),
    dec_max_current_meter         = COALESCE(sqlc.narg('max_current'), dec_max_current_meter),
    dec_contracted_power_kw_meter = COALESCE(sqlc.narg('contracted_power_kw'),
                                             dec_contracted_power_kw_meter),
    str_status_meter              = COALESCE(sqlc.narg('status'), str_status_meter)
WHERE uid_meter = sqlc.arg('id')
  AND dtm_deleted_at_meter IS NULL
RETURNING *;

-- name: SoftDeleteMeter :execrows
UPDATE meter
SET dtm_deleted_at_meter = now()
WHERE uid_meter = $1
  AND dtm_deleted_at_meter IS NULL;
