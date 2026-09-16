-- A análise da audiência produz três saídas: resumo, sugestão estratégica e a
-- lista de pontos críticos. As duas primeiras já tinham coluna; a terceira é
-- uma lista, e guardá-la como texto concatenado destruiria a estrutura que o
-- modelo devolveu (e que a interface precisa para renderizar item a item).
ALTER TABLE hearing.hearings
    ADD COLUMN IF NOT EXISTS pontos_criticos TEXT[] NOT NULL DEFAULT '{}';
