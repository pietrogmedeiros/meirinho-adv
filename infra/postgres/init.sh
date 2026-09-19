#!/bin/sh
# Roda uma vez, na criação do cluster (docker-entrypoint-initdb.d). Se o volume
# já existe, NÃO roda de novo: trocar APP_DB_PASSWORD depois exige alterar a
# senha no banco (ALTER ROLE) ou recriar o volume.
#
# meirinho_app é o papel usado pelos serviços. Ele não é dono de tabela e não
# tem BYPASSRLS — é o que faz as políticas de Row Level Security valerem.
set -e
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  -v senha="${APP_DB_PASSWORD:-meirinho_app}" <<'SQL'
CREATE ROLE meirinho_app LOGIN PASSWORD :'senha';
SQL
