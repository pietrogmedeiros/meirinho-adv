-- Roda uma vez, na criação do cluster (docker-entrypoint-initdb.d).
--
-- meirinho_app é o papel usado pelos serviços. Ele não é dono de tabela e não
-- tem BYPASSRLS — é o que faz as políticas de Row Level Security valerem.
CREATE ROLE meirinho_app LOGIN PASSWORD 'meirinho_app';
