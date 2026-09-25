-- +goose Up
-- Enums
CREATE TYPE meter_status AS ENUM (
    'OK',
    'ALERT',
    'CRITICAL'
    );

CREATE TYPE event_type AS ENUM (
    'OPERATIONAL_CHANGE',
    'SCHEDULED_OUTAGE',
    'DATA_QUALITY',
    'MAINTENANCE',
    'UNKNOWN'
    );

CREATE TYPE analysis_status AS ENUM (
    'PENDING',
    'RUNNING',
    'COMPLETED',
    'FAILED'
    );

CREATE TYPE anomaly_type AS ENUM (
    'REAL_ANOMALY',
    'EXPLAINABLE_ANOMALY',
    'FALSE_POSITIVE',
    'DATA_QUALITY'
    );

CREATE TYPE anomaly_severity AS ENUM (
    'LOW',
    'MEDIUM',
    'HIGH'
    );

CREATE TYPE anomaly_status AS ENUM (
    'OPEN',
    'INVESTIGATING',
    'RESOLVED',
    'DISMISSED'
    );

--Users
CREATE TABLE app_user
(
    uid_user          UUID PRIMARY KEY      DEFAULT gen_random_uuid(),
    num_id_user       INTEGER GENERATED ALWAYS AS IDENTITY NOT NULL UNIQUE,
    str_email_user    VARCHAR(255) NOT NULL UNIQUE,
    str_password_user VARCHAR(255) NOT NULL,
    str_name_user     VARCHAR(120) NOT NULL,
    dtm_created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    dtm_updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Meters
CREATE TABLE meter
(
    uid_meter                     UUID PRIMARY KEY          DEFAULT gen_random_uuid(),
    num_id_meter                  INTEGER GENERATED ALWAYS AS IDENTITY NOT NULL UNIQUE,
    str_code_meter                VARCHAR(20)      NOT NULL,
    str_name_meter                VARCHAR(120)     NOT NULL,
    str_location_meter            VARCHAR(200)     NOT NULL DEFAULT '',
    str_sector_meter              VARCHAR(60)      NOT NULL DEFAULT 'INDUSTRIAL',
    dec_nominal_voltage_meter     DOUBLE PRECISION NOT NULL DEFAULT 220,
    dec_max_current_meter         DOUBLE PRECISION,
    dec_contracted_power_kw_meter DOUBLE PRECISION,
    str_status_meter              meter_status     NOT NULL DEFAULT 'OK',
    dtm_deleted_at_meter          TIMESTAMPTZ,
    dtm_created_at                TIMESTAMPTZ      NOT NULL DEFAULT now(),
    dtm_updated_at                TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CONSTRAINT ck_meter_nominal_voltage CHECK (dec_nominal_voltage_meter > 0)
);

CREATE UNIQUE INDEX idx_meter_code_active ON meter (str_code_meter) WHERE dtm_deleted_at_meter IS NULL;

-- Readings
CREATE TABLE reading
(
    num_id_reading              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid_meter                   UUID             NOT NULL REFERENCES meter (uid_meter) ON DELETE CASCADE,
    dtm_timestamp_reading       TIMESTAMPTZ      NOT NULL,
    dec_consumption_kwh_reading DOUBLE PRECISION NOT NULL,
    dec_voltage_reading         DOUBLE PRECISION NOT NULL,
    dec_current_reading         DOUBLE PRECISION NOT NULL,
    dec_power_factor_reading    DOUBLE PRECISION NOT NULL,
    str_status_reading          VARCHAR(20)      NOT NULL DEFAULT 'OK',
    dtm_created_at              TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CONSTRAINT uq_reading_meter_timestamp UNIQUE (uid_meter, dtm_timestamp_reading)
);

-- Events
CREATE TABLE event
(
    uid_event             UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    num_id_event          INTEGER GENERATED ALWAYS AS IDENTITY NOT NULL UNIQUE,
    uid_meter             UUID        NOT NULL REFERENCES meter (uid_meter) ON DELETE CASCADE,
    dtm_timestamp_event   TIMESTAMPTZ NOT NULL,
    str_type_event        event_type  NOT NULL,
    str_description_event TEXT        NOT NULL DEFAULT '',
    dtm_created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    dtm_updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_event_meter_timestamp_type UNIQUE (uid_meter, dtm_timestamp_event, str_type_event)
);

-- Analysis
CREATE TABLE analysis
(
    uid_analysis               UUID PRIMARY KEY         DEFAULT gen_random_uuid(),
    num_id_analysis            INTEGER GENERATED ALWAYS AS IDENTITY NOT NULL UNIQUE,
    str_status_analysis        analysis_status NOT NULL DEFAULT 'PENDING',
    str_current_step_analysis  VARCHAR(40),
    num_progress_analysis      SMALLINT        NOT NULL DEFAULT 0,
    str_provider_analysis      VARCHAR(40),
    num_meters_analyzed        INTEGER         NOT NULL DEFAULT 0,
    num_anomalies_analysis     INTEGER         NOT NULL DEFAULT 0,
    num_high_priority_analysis INTEGER         NOT NULL DEFAULT 0,
    str_error_analysis         TEXT,
    uid_user                   UUID            REFERENCES app_user (uid_user) ON DELETE SET NULL,
    dtm_started_at_analysis    TIMESTAMPTZ,
    dtm_finished_at_analysis   TIMESTAMPTZ,
    dtm_created_at             TIMESTAMPTZ     NOT NULL DEFAULT now(),
    dtm_updated_at             TIMESTAMPTZ     NOT NULL DEFAULT now(),
    CONSTRAINT chk_analysis_progress CHECK (num_progress_analysis BETWEEN 0 AND 100)
);

-- Anomalies
CREATE TABLE anomaly
(
    uid_anomaly                    UUID PRIMARY KEY          DEFAULT gen_random_uuid(),
    num_id_anomaly                 INTEGER GENERATED ALWAYS AS IDENTITY NOT NULL UNIQUE,
    uid_analysis                   UUID             NOT NULL REFERENCES analysis (uid_analysis) ON DELETE CASCADE,
    uid_meter                      UUID             NOT NULL REFERENCES meter (uid_meter) ON DELETE CASCADE,
    uid_event                      UUID             REFERENCES event (uid_event) ON DELETE SET NULL,
    str_type_anomaly               anomaly_type     NOT NULL,
    str_severity_anomaly           anomaly_severity NOT NULL,
    str_status_anomaly             anomaly_status   NOT NULL DEFAULT 'OPEN',
    dec_confidence_anomaly         DOUBLE PRECISION NOT NULL,
    dec_priority_score_anomaly     DOUBLE PRECISION NOT NULL DEFAULT 0,
    dec_baseline_kwh_anomaly       DOUBLE PRECISION,
    dec_current_kwh_anomaly        DOUBLE PRECISION,
    dec_variation_pct_anomaly      DOUBLE PRECISION,
    dtm_window_start_anomaly       TIMESTAMPTZ,
    dtm_window_end_anomaly         TIMESTAMPTZ,
    str_reason_anomaly             TEXT             NOT NULL,
    str_recommended_action_anomaly TEXT             NOT NULL,
    arr_changed_variables_anomaly  TEXT[]           NOT NULL DEFAULT '{}',
    json_evidence_anomaly          JSONB            NOT NULL DEFAULT '{}'::jsonb,
    str_resolution_note_anomaly    TEXT,
    dtm_detected_at_anomaly        TIMESTAMPTZ      NOT NULL DEFAULT now(),
    dtm_created_at                 TIMESTAMPTZ      NOT NULL DEFAULT now(),
    dtm_updated_at                 TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CONSTRAINT chk_anomaly_confidence CHECK (dec_confidence_anomaly BETWEEN 0 AND 1)
);

CREATE INDEX idx_event_meter_timestamp ON event (uid_meter, dtm_timestamp_event);
CREATE INDEX idx_anomaly_analysis ON anomaly (uid_analysis);
CREATE INDEX idx_anomaly_meter ON anomaly (uid_meter);
CREATE INDEX idx_anomaly_priority ON anomaly (dec_priority_score_anomaly DESC);


-- Triggers
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION fn_set_updated_at()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.dtm_updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_app_user_updated_at
    BEFORE UPDATE
    ON app_user
    FOR EACH ROW
EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_meter_updated_at
    BEFORE UPDATE
    ON meter
    FOR EACH ROW
EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_event_updated_at
    BEFORE UPDATE
    ON event
    FOR EACH ROW
EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_analysis_updated_at
    BEFORE UPDATE
    ON analysis
    FOR EACH ROW
EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_anomaly_updated_at
    BEFORE UPDATE
    ON anomaly
    FOR EACH ROW
EXECUTE FUNCTION fn_set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS anomaly;
DROP TABLE IF EXISTS analysis;
DROP TABLE IF EXISTS event;
DROP TABLE IF EXISTS reading;
DROP TABLE IF EXISTS meter;
DROP TABLE IF EXISTS app_user;
DROP FUNCTION IF EXISTS fn_set_updated_at();
DROP TYPE IF EXISTS anomaly_status;
DROP TYPE IF EXISTS anomaly_severity;
DROP TYPE IF EXISTS anomaly_type;
DROP TYPE IF EXISTS analysis_status;
DROP TYPE IF EXISTS event_type;
DROP TYPE IF EXISTS meter_status;
