package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/config"
)

var errCanalDesligado = errors.New("canal desligado por feature flag")

// Canal é uma via de entrega. O in-app não é um canal de saída de verdade — a
// notificação já está no banco e a interface lê de lá —, mas modelá-lo junto
// mantém um caminho só para "notificar o advogado".
type Canal interface {
	Enviar(ctx context.Context, n notificacao) error
	Tipo() domain.TipoNotificacao
}

// despachante resolve por quais canais uma notificação sai.
//
// O in-app está sempre ligado: é o registro. Email e WhatsApp são aditivos e
// desligados por padrão — o MVP não tem provedor de envio contratado, e um
// canal que falha silenciosamente é pior que um canal ausente.
type despachante struct {
	canais []Canal
	log    *slog.Logger
}

func novoDespachante(log *slog.Logger) *despachante {
	d := &despachante{log: log}

	if config.Bool("NOTIFY_EMAIL_ENABLED", false) {
		d.canais = append(d.canais, &canalLog{tipo: domain.NotifEmail, log: log})
	}
	if config.Bool("NOTIFY_WHATSAPP_ENABLED", false) {
		d.canais = append(d.canais, &canalLog{tipo: domain.NotifWhatsApp, log: log})
	}
	return d
}

// Despachar nunca falha o processamento do evento: a notificação já está
// persistida e visível in-app. Um canal externo fora do ar não pode fazer a
// mensagem ser reprocessada e duplicar o registro.
func (d *despachante) Despachar(ctx context.Context, n notificacao) {
	for _, c := range d.canais {
		if err := c.Enviar(ctx, n); err != nil {
			d.log.Error("falha ao enviar notificação",
				"canal", c.Tipo(), "notificacao_id", n.ID, "err", err)
		}
	}
}

// canalLog é o stand-in dos canais externos enquanto não há provedor: registra
// o que teria sido enviado, com o mesmo conteúdo. Trocar por SMTP ou pela API
// do WhatsApp é implementar Canal e registrar no lugar dele.
type canalLog struct {
	tipo domain.TipoNotificacao
	log  *slog.Logger
}

func (c *canalLog) Tipo() domain.TipoNotificacao { return c.tipo }

func (c *canalLog) Enviar(ctx context.Context, n notificacao) error {
	c.log.Info("envio simulado de notificação",
		"canal", c.tipo, "titulo", n.Titulo, "tenant_id", n.TenantID)
	return nil
}
