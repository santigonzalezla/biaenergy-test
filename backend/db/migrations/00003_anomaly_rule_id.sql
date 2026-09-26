-- +goose Up

-- Regla del motor de IA que clasificó la anomalía (R1_MEASUREMENT_FAULT ... R6_EQUIPMENT_FAULT):
-- trazabilidad de cada conclusión hasta la regla que la produjo.
-- El DEFAULT solo cubre filas previas; se elimina para que toda inserción nueva deba indicar su regla.
ALTER TABLE anomaly
    ADD COLUMN str_rule_id_anomaly VARCHAR(40) NOT NULL DEFAULT '';

ALTER TABLE anomaly
    ALTER COLUMN str_rule_id_anomaly DROP DEFAULT;

-- +goose Down
ALTER TABLE anomaly
    DROP COLUMN IF EXISTS str_rule_id_anomaly;
