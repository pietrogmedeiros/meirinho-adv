-- Um schema por domínio, um banco só. A separação lógica já existe agora; se
-- um domínio precisar de banco próprio depois, o schema vira a fronteira natural.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS hearing;
CREATE SCHEMA IF NOT EXISTS process;
CREATE SCHEMA IF NOT EXISTS notification;

-- ---------------------------------------------------------------------------
-- auth: um advogado == um tenant no MVP.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auth.tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome          TEXT        NOT NULL,
    oab           TEXT        NOT NULL,
    email         TEXT        NOT NULL,
    senha_hash    TEXT        NOT NULL,
    -- LGPD: retenção configurável por tenant. No MVP é só o campo (não há job
    -- de expurgo ainda), mas a política já nasce com o dado.
    retencao_dias INT         NOT NULL DEFAULT 365 CHECK (retencao_dias BETWEEN 30 AND 3650),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS tenants_email_uniq ON auth.tenants (lower(email));

-- ---------------------------------------------------------------------------
-- hearing: pipeline da audiência.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS hearing.hearings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES auth.tenants (id) ON DELETE CASCADE,
    titulo          TEXT        NOT NULL DEFAULT '',
    object_key      TEXT        NOT NULL,
    nome_arquivo    TEXT        NOT NULL DEFAULT '',
    area_do_direito TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'uploaded',
    transcript      TEXT,
    summary         TEXT,
    suggestion      TEXT,
    erro            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- tenant_id vem primeiro no índice: toda listagem filtra por ele.
CREATE INDEX IF NOT EXISTS hearings_tenant_created_idx
    ON hearing.hearings (tenant_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- process: processos monitorados e suas movimentações.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS process.monitored_processes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES auth.tenants (id) ON DELETE CASCADE,
    numero_cnj      TEXT        NOT NULL,
    titulo          TEXT        NOT NULL DEFAULT '',
    tribunal        TEXT        NOT NULL DEFAULT '',
    area_do_direito TEXT        NOT NULL DEFAULT 'civel',
    provider        TEXT        NOT NULL DEFAULT 'mock',
    provider_ref    TEXT        NOT NULL DEFAULT '',
    status          TEXT        NOT NULL DEFAULT 'ativo',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- O mesmo processo não entra duas vezes na carteira do mesmo advogado — mas
-- dois advogados podem monitorar o mesmo CNJ.
CREATE UNIQUE INDEX IF NOT EXISTS processes_tenant_cnj_uniq
    ON process.monitored_processes (tenant_id, numero_cnj);

CREATE TABLE IF NOT EXISTS process.movements (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL REFERENCES auth.tenants (id) ON DELETE CASCADE,
    process_id           UUID        NOT NULL REFERENCES process.monitored_processes (id) ON DELETE CASCADE,
    -- Chave de idempotência: o provider pode reenviar o mesmo evento.
    provider_event_id    TEXT        NOT NULL,
    descricao            TEXT        NOT NULL,
    data                 TIMESTAMPTZ NOT NULL,
    urgencia_classificada TEXT,
    sugestao             TEXT,
    prazo_dias           INT,
    -- Marca quando a urgência veio da regra conservadora e não do classificador.
    fallback_aplicado    BOOLEAN     NOT NULL DEFAULT FALSE,
    notificado_em        TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Idempotência garantida pelo banco, não por checagem otimista na aplicação.
CREATE UNIQUE INDEX IF NOT EXISTS movements_provider_event_uniq
    ON process.movements (process_id, provider_event_id);

CREATE INDEX IF NOT EXISTS movements_tenant_data_idx
    ON process.movements (tenant_id, data DESC);

-- ---------------------------------------------------------------------------
-- notification
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification.notifications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES auth.tenants (id) ON DELETE CASCADE,
    tipo          TEXT        NOT NULL CHECK (tipo IN ('in_app', 'email', 'whatsapp')),
    categoria     TEXT        NOT NULL DEFAULT 'geral',
    titulo        TEXT        NOT NULL,
    corpo         TEXT        NOT NULL DEFAULT '',
    urgencia      TEXT,
    referencia_id UUID,
    lida          BOOLEAN     NOT NULL DEFAULT FALSE,
    enviado_em    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notifications_tenant_created_idx
    ON notification.notifications (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS notifications_tenant_nao_lidas_idx
    ON notification.notifications (tenant_id) WHERE NOT lida;
