-- Isolamento por tenant reforçado no banco.
--
-- A regra "toda query nasce filtrada por tenant_id" não pode depender de o
-- desenvolvedor lembrar do WHERE. Com ROW LEVEL SECURITY, uma query sem filtro
-- simplesmente não devolve linha de outro tenant: o Postgres aplica a política
-- comparando tenant_id com a variável de sessão app.current_tenant, que só é
-- setada por db.TenantTx.
--
-- Importante: o dono da tabela ignora RLS por padrão. Por isso as aplicações
-- conectam como meirinho_app (papel sem BYPASSRLS, criado em infra/postgres/init.sql),
-- enquanto as migrations rodam como o dono.

GRANT USAGE ON SCHEMA auth, hearing, process, notification TO meirinho_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA auth, hearing, process, notification TO meirinho_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA auth, hearing, process, notification
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO meirinho_app;

-- auth.tenants fica fora do RLS: cadastro e login acontecem antes de existir
-- um tenant corrente. O acesso ali é restrito pelas queries do auth-service.
ALTER TABLE hearing.hearings               ENABLE ROW LEVEL SECURITY;
ALTER TABLE process.monitored_processes    ENABLE ROW LEVEL SECURITY;
ALTER TABLE process.movements              ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification.notifications     ENABLE ROW LEVEL SECURITY;

-- FORCE para que a política valha inclusive para o dono da tabela.
ALTER TABLE hearing.hearings               FORCE ROW LEVEL SECURITY;
ALTER TABLE process.monitored_processes    FORCE ROW LEVEL SECURITY;
ALTER TABLE process.movements              FORCE ROW LEVEL SECURITY;
ALTER TABLE notification.notifications     FORCE ROW LEVEL SECURITY;

-- current_setting(..., true) devolve NULL quando a variável não foi setada;
-- NULLIF + cast garantem que "sem tenant setado" signifique "nenhuma linha".
CREATE POLICY tenant_isolado ON hearing.hearings
    USING      (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

CREATE POLICY tenant_isolado ON process.monitored_processes
    USING      (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

CREATE POLICY tenant_isolado ON process.movements
    USING      (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

CREATE POLICY tenant_isolado ON notification.notifications
    USING      (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
