// Package llm é a fronteira com o modelo de linguagem.
//
// Mora em platform, e não dentro de um serviço, porque dois consumidores
// diferentes precisam do mesmo contrato: o analysis-service analisa audiências
// transcritas e o classification-worker classifica movimentações processuais.
// Audiência e movimentação são fontes de dado distintas lidas pelo mesmo
// padrão — contexto da norma da área + pedido de saída estruturada.
package llm

import (
	"context"

	"github.com/pietromedeiros/meirinho/internal/domain"
)

// AnaliseAudiencia é o produto da análise de uma audiência transcrita.
type AnaliseAudiencia struct {
	Resumo              string   `json:"resumo"`
	SugestaoEstrategica string   `json:"sugestao_estrategica"`
	PontosCriticos      []string `json:"pontos_criticos"`
}

// ClassificacaoMovimentacao é o produto da leitura de uma movimentação.
//
// Confianca é o campo que sustenta a regra de fallback conservador do
// classification-worker: sem ela, "sem urgência" e "não sei" seriam
// indistinguíveis — e o segundo caso precisa virar alerta.
type ClassificacaoMovimentacao struct {
	Urgencia  domain.Urgencia `json:"urgencia"`
	PrazoDias *int            `json:"prazo_dias"`
	Sugestao  string          `json:"sugestao"`
	Confianca float64         `json:"confianca"`
	Motivo    string          `json:"motivo"`
}

type Analyzer interface {
	AnalisarAudiencia(ctx context.Context, transcricao string, area domain.AreaDoDireito) (*AnaliseAudiencia, error)
	ClassificarMovimentacao(ctx context.Context, descricao string, area domain.AreaDoDireito) (*ClassificacaoMovimentacao, error)
	Nome() string
}
