-- name: CreateImportBatch :one
-- Una importación exitosa crea el lote al inicio de su transacción (las filas lo referencian) y un fallo lo crea ya cerrado.
INSERT INTO import_batch (str_kind_import_batch, str_status_import_batch, str_file_name_import_batch,
                          num_file_size_import_batch, str_checksum_import_batch, num_rows_import_batch,
                          str_error_code_import_batch, str_error_import_batch, uid_user)
VALUES (sqlc.arg('kind'), sqlc.arg('status'), sqlc.arg('file_name'), sqlc.arg('file_size'), sqlc.arg('checksum'),
        sqlc.arg('rows'), sqlc.narg('error_code'), sqlc.narg('error'), sqlc.narg('user_id'))
RETURNING *;

-- name: CompleteImportBatch :exec
UPDATE import_batch
SET num_inserted_import_batch        = sqlc.arg('inserted'),
    num_skipped_import_batch         = sqlc.arg('skipped'),
    num_meters_created_import_batch  = sqlc.arg('meters_created'),
    num_meters_restored_import_batch = sqlc.arg('meters_restored')
WHERE uid_import_batch = sqlc.arg('id');

-- name: ListImportBatches :many
SELECT b.*, u.str_name_user
FROM import_batch b
         LEFT JOIN app_user u ON u.uid_user = b.uid_user
ORDER BY b.dtm_created_at DESC
LIMIT sqlc.arg('page_limit');

-- name: FindCompletedImportByChecksum :one
-- Detecta que el mismo archivo ya se importó con éxito: se avisa al usuario, pero se permite reimportar (es idempotente).
SELECT uid_import_batch, dtm_created_at
FROM import_batch
WHERE str_kind_import_batch = sqlc.arg('kind')
  AND str_checksum_import_batch = sqlc.arg('checksum')
  AND str_status_import_batch = 'COMPLETED'
ORDER BY dtm_created_at DESC
LIMIT 1;
