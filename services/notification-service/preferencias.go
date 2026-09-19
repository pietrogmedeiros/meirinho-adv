package main

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/authn"
	"github.com/pietromedeiros/meirinho/internal/platform/httpx"
	"github.com/pietromedeiros/meirinho/internal/platform/logging"
)

type preferencias struct {
	UrgenciaMinima    domain.Urgencia `json:"urgencia_minima"`
	AlertarAudiencias bool            `json:"alertar_audiencias"`
	CanalEmail        bool            `json:"canal_email"`
	CanalWhatsApp     bool            `json:"canal_whatsapp"`
	Telefone          string          `json:"telefone"`
}

// preferenciasPadrao valem enquanto o advogado não salvou nada. A urgência
// mínima padrão é a global do serviço (NOTIFY_URGENCIA_MINIMA).
func (a *api) preferenciasPadrao() preferencias {
	return preferencias{UrgenciaMinima: a.urgenciaMinima, AlertarAudiencias: true}
}

func (s *store) preferencias(ctx context.Context, tenantID string, padrao preferencias) (preferencias, error) {
	p := padrao
	err := s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT urgencia_minima, alertar_audiencias, canal_email, canal_whatsapp, telefone
			  FROM notification.preferencias`,
		).Scan(&p.UrgenciaMinima, &p.AlertarAudiencias, &p.CanalEmail, &p.CanalWhatsApp, &p.Telefone)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return padrao, nil
	}
	return p, err
}

func (s *store) salvarPreferencias(ctx context.Context, tenantID string, p preferencias) error {
	return s.pool.TenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO notification.preferencias
			       (tenant_id, urgencia_minima, alertar_audiencias, canal_email, canal_whatsapp, telefone, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, now())
			ON CONFLICT (tenant_id) DO UPDATE SET
			       urgencia_minima = EXCLUDED.urgencia_minima,
			       alertar_audiencias = EXCLUDED.alertar_audiencias,
			       canal_email = EXCLUDED.canal_email,
			       canal_whatsapp = EXCLUDED.canal_whatsapp,
			       telefone = EXCLUDED.telefone,
			       updated_at = now()`,
			tenantID, string(p.UrgenciaMinima), p.AlertarAudiencias, p.CanalEmail, p.CanalWhatsApp, p.Telefone)
		return err
	})
}

// prefsDoTenant é o que os consumidores usam: uma falha ao ler preferências
// não pode engolir o alerta, então cai nos padrões.
func (a *api) prefsDoTenant(ctx context.Context, tenantID string) preferencias {
	p, err := a.store.preferencias(ctx, tenantID, a.preferenciasPadrao())
	if err != nil {
		logging.From(ctx).Warn("ler preferências; usando padrões", "err", err, "tenant_id", tenantID)
		return a.preferenciasPadrao()
	}
	return p
}

type respPreferencias struct {
	preferencias
	// Canais que o servidor consegue de fato entregar. A interface mostra a
	// opção de todo jeito, mas avisa quando ela ainda não sai do servidor —
	// um toggle que não faz nada sem dizer isso é pior que nenhum.
	CanaisDisponiveis map[string]bool `json:"canais_disponiveis"`
}

func (a *api) lerPreferencias(w http.ResponseWriter, r *http.Request) {
	p := a.prefsDoTenant(r.Context(), authn.TenantDo(r.Context()))
	httpx.JSON(w, http.StatusOK, respPreferencias{p, a.despachante.disponiveis()})
}

var reTelefone = regexp.MustCompile(`^\+?[0-9]{10,15}$`)

func (a *api) salvarPreferencias(w http.ResponseWriter, r *http.Request) {
	var p preferencias
	if err := httpx.Decode(r, &p); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "corpo_invalido", "JSON inválido")
		return
	}
	if _, ok := domain.ParseUrgencia(string(p.UrgenciaMinima)); !ok {
		httpx.Fail(w, http.StatusBadRequest, "urgencia_invalida", "urgência mínima inválida")
		return
	}
	tel := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(p.Telefone)
	if p.CanalWhatsApp && !reTelefone.MatchString(tel) {
		httpx.Fail(w, http.StatusBadRequest, "telefone_invalido", "informe um celular com DDD para receber por WhatsApp")
		return
	}
	p.Telefone = tel

	tenantID := authn.TenantDo(r.Context())
	if err := a.store.salvarPreferencias(r.Context(), tenantID, p); err != nil {
		logging.From(r.Context()).Error("salvar preferências", "err", err)
		httpx.Fail(w, http.StatusInternalServerError, "erro_interno", "não foi possível salvar")
		return
	}
	httpx.JSON(w, http.StatusOK, respPreferencias{p, a.despachante.disponiveis()})
}
