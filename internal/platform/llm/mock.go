package llm

import (
	"context"
	"strings"
	"time"

	"github.com/pietromedeiros/meirinho/internal/domain"
)

// mockAnalyzer devolve resultado plausível sem chamar a API.
//
// É o padrão do compose: `docker compose up` tem que funcionar numa máquina sem
// ANTHROPIC_API_KEY, senão o pipeline inteiro fica intestável localmente. Quem
// tem chave liga ANALYZER=claude.
type mockAnalyzer struct{ atraso time.Duration }

func (m *mockAnalyzer) Nome() string { return "mock" }

func (m *mockAnalyzer) esperar(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(m.atraso):
		return nil
	}
}

func (m *mockAnalyzer) AnalisarAudiencia(ctx context.Context, transcricao string, area domain.AreaDoDireito) (*AnaliseAudiencia, error) {
	if err := m.esperar(ctx); err != nil {
		return nil, err
	}
	return &AnaliseAudiencia{
		Resumo: "[mock] Análise simulada de audiência em " + area.Norma() +
			". Transcrição recebida com " + itoa(len(transcricao)) + " caracteres.",
		SugestaoEstrategica: "[mock] Defina ANALYZER=claude e ANTHROPIC_API_KEY para receber análise real.",
		PontosCriticos:      []string{"[mock] nenhum ponto crítico real foi avaliado"},
	}, nil
}

// ClassificarMovimentacao do mock usa heurística de palavra-chave, não sorteio.
// Assim o fallback conservador do classification-worker fica exercitável
// localmente: descrição genérica cai em confiança baixa e vira alerta.
func (m *mockAnalyzer) ClassificarMovimentacao(ctx context.Context, descricao string, area domain.AreaDoDireito) (*ClassificacaoMovimentacao, error) {
	if err := m.esperar(ctx); err != nil {
		return nil, err
	}
	d := strings.ToLower(descricao)
	prazo := 15

	switch {
	case contemAlguma(d, "intima", "prazo", "cite-se", "manifeste", "contesta", "recurso", "sentença"):
		return &ClassificacaoMovimentacao{
			Urgencia: domain.UrgenciaAlta, PrazoDias: &prazo, Confianca: 0.9,
			Sugestao: "[mock] Verificar o prazo e protocolar a manifestação.",
			Motivo:   "[mock] termo indicativo de prazo encontrado na descrição",
		}, nil
	case contemAlguma(d, "despacho", "decisão", "audiência", "perícia"):
		return &ClassificacaoMovimentacao{
			Urgencia: domain.UrgenciaMedia, Confianca: 0.8,
			Sugestao: "[mock] Acompanhar o desdobramento.",
			Motivo:   "[mock] movimentação relevante sem prazo explícito",
		}, nil
	case contemAlguma(d, "juntada", "publicado", "distribuído", "autuado"):
		return &ClassificacaoMovimentacao{
			Urgencia: domain.UrgenciaBaixa, Confianca: 0.75,
			Sugestao: "[mock] Apenas registrar.", Motivo: "[mock] ato cartorário",
		}, nil
	default:
		// Confiança baixa de propósito: dispara o fallback conservador.
		return &ClassificacaoMovimentacao{
			Urgencia: domain.UrgenciaNenhuma, Confianca: 0.2,
			Sugestao: "[mock] Ler a movimentação manualmente.",
			Motivo:   "[mock] descrição não reconhecida pela heurística",
		}, nil
	}
}

func contemAlguma(s string, termos ...string) bool {
	for _, t := range termos {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
