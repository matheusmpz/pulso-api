-- =====================================================================
-- PULAPI-023 — cor dos dashboards
--
-- Acrescenta a coluna "cor" em dashboards. O valor é o NOME de uma cor da
-- paleta do Tailwind (emerald, violet ou orange) que o front usa nos
-- gráficos daquele dashboard. Nulo = o front usa a cor padrão.
--
-- Idempotente: pode rodar mais de uma vez sem erro e sem sobrescrever uma
-- cor já escolhida (o UPDATE só preenche onde cor IS NULL).
--
-- Bancos existentes: aplique manualmente, ex.:
--   docker compose exec -T db psql -v ON_ERROR_STOP=1 -U <DB_USER> -d <DB_NAME> < db/migrations/PULAPI-023_script.sql
-- Bancos novos no Docker: entra sozinho (db/docker-init-migrations.sh).
-- =====================================================================

BEGIN;

ALTER TABLE dashboards ADD COLUMN IF NOT EXISTS cor TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_dashboards_cor' AND conrelid = 'dashboards'::regclass
    ) THEN
        ALTER TABLE dashboards
            ADD CONSTRAINT ck_dashboards_cor CHECK (cor IS NULL OR cor IN ('emerald', 'violet', 'orange'));
    END IF;
END $$;

UPDATE dashboards d
SET cor = v.cor
FROM (VALUES ('vendas', 'emerald'), ('projetos', 'violet'), ('ia', 'orange')) AS v(chave, cor)
WHERE d.chave = v.chave
  AND d.cor IS NULL;

COMMIT;
