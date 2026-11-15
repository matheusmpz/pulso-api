#!/bin/sh
# =====================================================================
# Executa os scripts de db/migrations/ num banco NOVO do Docker.
#
# O Postgres do container roda o que está em /docker-entrypoint-initdb.d
# só na primeira criação do volume, em ordem alfabética:
#   01-schema.sql → 02-seed.sql → 03-migrations.sh (este script)
# Este script percorre as migrations (montadas em /pulso-migrations, só
# leitura) em ordem alfabética (PULAPI-023, PULAPI-024, ...) e executa cada
# uma. Qualquer erro interrompe a criação do banco.
#
# Bancos já existentes não passam por aqui: os scripts novos são aplicados
# manualmente (veja o README).
# =====================================================================
set -e

PASTA=/pulso-migrations

for arquivo in "$PASTA"/*.sql; do
    # Sem nenhum .sql, o padrão volta literal: não há o que executar
    [ -e "$arquivo" ] || continue
    echo "Migration: $(basename "$arquivo")"
    psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f "$arquivo"
done
