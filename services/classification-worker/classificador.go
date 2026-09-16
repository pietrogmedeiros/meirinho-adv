package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/llm"
)

// classificador envolve o analyzer com a regra que define o comportamento do
// produto: na dúvida, alerta.
//
// O risco dos dois lados não é simétrico. Um alerta desnecessário custa ao
// advogado trinta segundos de leitura; uma intimação classificada como
// "nenhuma" custa um prazo perdido. Então toda incerteza é resolvida para
// cima — nunca para baixo.
type classificador struct {
	analyzer     llm.Analyzer
	piso         domain.Urgencia // urgência mínima quando não se pode confiar no resultado
	minConfianca float64
	tentativas   int
	log          *slog.Logger
}

// Classificar devolve sempre uma classificação utilizável. O bool indica se o
// fallback foi aplicado — o que viaja no evento e na coluna fallback_aplicado
// para que dê para medir depois com que frequência o classificador hesita.
func (c *classificador) Classificar(ctx context.Context, ev domain.MovementReceived) (*llm.ClassificacaoMovimentacao, bool) {
	res, err := c.comRetry(ctx, ev.Descricao, ev.Area)
	if err != nil {
		// Sem classificação nenhuma. A mensagem ficaria pendente no stream
		// para sempre (não há reivindicação de pendentes), e uma movimentação
		// engolida em silêncio é o pior desfecho possível: entrega o alerta
		// com a urgência do piso e diz na sugestão que a leitura é manual.
		c.log.Error("classificação falhou, aplicando piso conservador",
			"err", err, "movement_id", ev.MovementID)
		return &llm.ClassificacaoMovimentacao{
			Urgencia:  c.piso,
			Sugestao:  "Não foi possível classificar automaticamente — leia a movimentação e avalie o prazo manualmente.",
			Motivo:    fmt.Sprintf("classificador indisponível: %v", err),
			Confianca: 0,
		}, true
	}

	motivo := ""
	switch {
	case !res.Urgencia.Valida():
		motivo = fmt.Sprintf("urgência %q fora do domínio conhecido", res.Urgencia)
	case res.Confianca < c.minConfianca:
		motivo = fmt.Sprintf("confiança %.2f abaixo do mínimo de %.2f", res.Confianca, c.minConfianca)
	default:
		return res, false // classificação confiável, vale como está
	}

	// Elevar, nunca rebaixar: se o modelo já classificou acima do piso mesmo
	// com pouca confiança, a classificação dele é mantida.
	if res.Urgencia.Nivel() < c.piso.Nivel() {
		c.log.Warn("elevando urgência por fallback conservador",
			"de", res.Urgencia, "para", c.piso, "motivo", motivo)
		res.Urgencia = c.piso
	}
	res.Motivo = res.Motivo + " [fallback: " + motivo + "]"
	return res, true
}

func (c *classificador) comRetry(ctx context.Context, descricao string, area domain.AreaDoDireito) (*llm.ClassificacaoMovimentacao, error) {
	var err error
	for i := 0; i < c.tentativas; i++ {
		var res *llm.ClassificacaoMovimentacao
		res, err = c.analyzer.ClassificarMovimentacao(ctx, descricao, area)
		if err == nil {
			return res, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, err
}
