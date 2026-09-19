-- Preferências de notificação por advogado. Mora no schema do notification-
-- service porque é ele quem decide o que vira alerta e por qual canal; ausência
-- de linha significa "padrões" (tudo a partir de baixa urgência, só in-app).
CREATE TABLE IF NOT EXISTS notification.preferencias (
    tenant_id       UUID PRIMARY KEY REFERENCES auth.tenants (id) ON DELETE CASCADE,
    -- Abaixo deste nível a movimentação entra no histórico sem alerta. O
    -- fallback do classificador continua furando o filtro.
    urgencia_minima TEXT        NOT NULL DEFAULT 'baixa'
                    CHECK (urgencia_minima IN ('alta', 'media', 'baixa', 'nenhuma')),
    alertar_audiencias BOOLEAN  NOT NULL DEFAULT TRUE,
    canal_email     BOOLEAN     NOT NULL DEFAULT FALSE,
    canal_whatsapp  BOOLEAN     NOT NULL DEFAULT FALSE,
    telefone        TEXT        NOT NULL DEFAULT '',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE notification.preferencias ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification.preferencias FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolado ON notification.preferencias
    USING      (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
