package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/events"
)

// poller é o que faz o monitoramento existir: consulta o provider de tempo em
// tempo e injeta no barramento o que ainda não tinha sido visto.
//
// Consulta por tenant, não uma varredura global: a lista de processos está sob
// RLS, então o único jeito de enxergá-la é abrir uma transação com o tenant
// setado. O efeito colateral é bom — um tenant com problema não interrompe a
// varredura dos outros.
type poller struct {
	store     *store
	bus       *events.Bus
	provider  Provider
	intervalo time.Duration
	log       *slog.Logger
}

func (p *poller) Rodar(ctx context.Context) {
	// Primeira passada imediata: subir o serviço e esperar o intervalo inteiro
	// para ver qualquer sinal de vida tornaria o desenvolvimento penoso.
	p.passada(ctx)

	t := time.NewTicker(p.intervalo)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			p.log.Info("poller encerrado")
			return
		case <-t.C:
			p.passada(ctx)
		}
	}
}

func (p *poller) passada(ctx context.Context) {
	inicio := time.Now()
	tenants, err := p.store.tenants(ctx)
	if err != nil {
		p.log.Error("poller: listar tenants", "err", err)
		return
	}

	var processos, novas int
	for _, tenantID := range tenants {
		if ctx.Err() != nil {
			return
		}
		n, m := p.passadaTenant(ctx, tenantID)
		processos += n
		novas += m
	}

	p.log.Info("passada do poller concluída",
		"tenants", len(tenants), "processos", processos,
		"movimentacoes_novas", novas, "duracao", time.Since(inicio).String())
}

func (p *poller) passadaTenant(ctx context.Context, tenantID string) (int, int) {
	ativos, err := p.store.ativosDoTenant(ctx, tenantID)
	if err != nil {
		p.log.Error("poller: listar processos do tenant", "err", err, "tenant_id", tenantID)
		return 0, 0
	}

	novas := 0
	for _, proc := range ativos {
		if ctx.Err() != nil {
			return len(ativos), novas
		}
		novas += p.sincronizar(ctx, tenantID, proc)
	}
	return len(ativos), novas
}

// sincronizar consulta um processo e devolve quantas movimentações novas
// entraram. Erro em um processo é registrado e não aborta os demais: o provider
// pode falhar para um CNJ específico.
func (p *poller) sincronizar(ctx context.Context, tenantID string, proc processoAtivo) int {
	log := p.log.With("tenant_id", tenantID, "process_id", proc.ID, "numero_cnj", proc.NumeroCNJ)

	movs, err := p.provider.Movimentacoes(ctx, proc.NumeroCNJ, proc.ProviderRef, time.Time{})
	if err != nil {
		log.Error("consultar provider", "err", err)
		return 0
	}

	novas := 0
	for _, mv := range movs {
		id, err := p.store.inserirMovimentacao(ctx, tenantID, proc.ID, mv)
		if err != nil {
			log.Error("gravar movimentação", "err", err, "event_id", mv.EventID)
			continue
		}
		if id == "" {
			continue // já conhecida: o provider reenviou
		}

		// Publica só depois de gravar. Se o publish falhar, a movimentação
		// existe no banco e aparece na API; o que se perde é a classificação
		// automática, não o dado.
		if err := p.bus.Publish(ctx, domain.StreamMovementReceived, domain.MovementReceived{
			MovementID: id,
			ProcessID:  proc.ID,
			TenantID:   tenantID,
			NumeroCNJ:  proc.NumeroCNJ,
			Descricao:  mv.Descricao,
			Data:       mv.Data,
			Area:       proc.Area,
		}); err != nil {
			log.Error("publicar movement.received", "err", err, "movement_id", id)
			continue
		}
		novas++
	}
	return novas
}
