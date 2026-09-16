package llm

import (
	"fmt"

	"github.com/pietromedeiros/meirinho/internal/domain"
)

// O sistema é o mesmo para as duas fontes de dado: a identidade do analista e
// a norma da área. Só a tarefa muda. Manter esse prefixo estável também é o que
// permite o cache de prompt do lado do provedor.
func promptSistema(area domain.AreaDoDireito) string {
	return fmt.Sprintf(`Você é assistente jurídico de um advogado autônomo brasileiro que atua em %s.

Analise o material sempre à luz da legislação e da praxe forense brasileira
aplicável a essa área. Escreva em português do Brasil, em linguagem técnica mas
direta — quem lê é advogado, não leigo.

Regras invioláveis:
- Não invente fatos, datas, valores ou dispositivos legais que não estejam no material.
- Quando o material for insuficiente para concluir algo, diga isso explicitamente
  em vez de preencher a lacuna com suposição.
- Prazos processuais só devem ser afirmados quando decorrerem do que está escrito.`, area.Norma())
}

func promptAudiencia(transcricao string) string {
	return fmt.Sprintf(`Abaixo está a transcrição de uma audiência.

Produza:
1. Um resumo do que ocorreu: partes, pedidos, provas mencionadas, decisões do juízo e encaminhamentos.
2. Uma sugestão estratégica para o advogado: próximos passos concretos, considerando o que foi decidido e o que ficou em aberto.
3. Os pontos críticos — riscos, prazos, contradições ou pendências que exigem atenção imediata.

TRANSCRIÇÃO:
"""
%s
"""`, transcricao)
}

func promptMovimentacao(descricao string) string {
	return fmt.Sprintf(`Abaixo está uma movimentação processual recém-publicada.

Classifique a urgência para o advogado responsável:
- "alta": abre prazo, exige providência do advogado, ou pode gerar preclusão/revelia/perda de direito.
- "media": relevante para a estratégia, mas sem prazo imediato.
- "baixa": informativa, acompanhar sem agir.
- "nenhuma": puramente cartorária, sem efeito prático.

Informe também:
- prazo_dias: o prazo em dias corridos/úteis que a movimentação abre, quando ele
  decorrer claramente do texto; caso contrário, null.
- sugestao: a providência recomendada, em uma ou duas frases.
- confianca: de 0 a 1, o quanto o texto é suficiente para você classificar com
  segurança. Se a descrição for truncada, genérica ou ambígua, use valor baixo —
  é preferível admitir incerteza a classificar errado.
- motivo: por que você chegou a essa urgência.

MOVIMENTAÇÃO:
"""
%s
"""`, descricao)
}

// Esquemas de saída estruturada — a API valida o formato, então o parse do
// lado de cá não precisa lidar com JSON malformado nem com texto solto.
var schemaAudiencia = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"resumo":               map[string]any{"type": "string"},
		"sugestao_estrategica": map[string]any{"type": "string"},
		"pontos_criticos": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
	},
	"required":             []string{"resumo", "sugestao_estrategica", "pontos_criticos"},
	"additionalProperties": false,
}

var schemaMovimentacao = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"urgencia": map[string]any{
			"type": "string",
			"enum": []string{"alta", "media", "baixa", "nenhuma"},
		},
		"prazo_dias": map[string]any{"type": []string{"integer", "null"}},
		"sugestao":   map[string]any{"type": "string"},
		"confianca":  map[string]any{"type": "number", "minimum": 0, "maximum": 1},
		"motivo":     map[string]any{"type": "string"},
	},
	"required":             []string{"urgencia", "prazo_dias", "sugestao", "confianca", "motivo"},
	"additionalProperties": false,
}
