-- +goose Up
-- Historial de archivos importados: quién subió qué archivo, cuándo y con qué resultado (incluidos los fallidos).
CREATE TYPE import_kind AS ENUM ('READINGS', 'EVENTS');
CREATE TYPE import_status AS ENUM ('COMPLETED', 'FAILED');

CREATE TABLE import_batch
(
    uid_import_batch                 UUID PRIMARY KEY       DEFAULT gen_random_uuid(),
    num_id_import_batch              INTEGER GENERATED ALWAYS AS IDENTITY NOT NULL UNIQUE,
    str_kind_import_batch            import_kind   NOT NULL,
    str_status_import_batch          import_status NOT NULL,
    str_file_name_import_batch       VARCHAR(255)  NOT NULL,
    num_file_size_import_batch       BIGINT        NOT NULL,
    str_checksum_import_batch        CHAR(64)      NOT NULL,
    num_rows_import_batch            INTEGER       NOT NULL DEFAULT 0,
    num_inserted_import_batch        INTEGER       NOT NULL DEFAULT 0,
    num_skipped_import_batch         INTEGER       NOT NULL DEFAULT 0,
    num_meters_created_import_batch  INTEGER       NOT NULL DEFAULT 0,
    num_meters_restored_import_batch INTEGER       NOT NULL DEFAULT 0,
    str_error_code_import_batch      VARCHAR(40),
    str_error_import_batch           TEXT,
    uid_user                         UUID          REFERENCES app_user (uid_user) ON DELETE SET NULL,
    dtm_created_at                   TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX idx_import_batch_created ON import_batch (dtm_created_at DESC);
CREATE INDEX idx_import_batch_checksum ON import_batch (str_kind_import_batch, str_checksum_import_batch)
    WHERE str_status_import_batch = 'COMPLETED';

-- Trazabilidad: el lote que insertó cada fila. Las filas omitidas por duplicadas conservan su lote original.
ALTER TABLE reading ADD COLUMN uid_import_batch UUID REFERENCES import_batch (uid_import_batch) ON DELETE SET NULL;
ALTER TABLE event ADD COLUMN uid_import_batch UUID REFERENCES import_batch (uid_import_batch) ON DELETE SET NULL;

CREATE INDEX idx_reading_import_batch ON reading (uid_import_batch);
CREATE INDEX idx_event_import_batch ON event (uid_import_batch);

-- +goose Down
DROP INDEX IF EXISTS idx_event_import_batch;
DROP INDEX IF EXISTS idx_reading_import_batch;

ALTER TABLE event DROP COLUMN IF EXISTS uid_import_batch;
ALTER TABLE reading DROP COLUMN IF EXISTS uid_import_batch;

DROP TABLE IF EXISTS import_batch;
DROP TYPE IF EXISTS import_status;
DROP TYPE IF EXISTS import_kind;
