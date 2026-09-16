package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/pietromedeiros/meirinho/internal/domain"
)

// claudeAnalyzer implementa Analyzer sobre a Claude API.
//
// Usa saída estruturada (output_config.format): o formato da resposta é
// validado pelo próprio serviço, o que elimina toda uma classe de bug de parse
// que normalmente aparece quando se pede "responda em JSON" no prompt.
type claudeAnalyzer struct {
	client anthropic.Client
	modelo string
}

func novoClaudeAnalyzer(apiKey, modelo string) *claudeAnalyzer {
	return &claudeAnalyzer{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		modelo: modelo,
	}
}

func (c *claudeAnalyzer) Nome() string { return "claude:" + c.modelo }

func (c *claudeAnalyzer) AnalisarAudiencia(ctx context.Context, transcricao string, area domain.AreaDoDireito) (*AnaliseAudiencia, error) {
	var out AnaliseAudiencia
	if err := c.pedir(ctx, area, promptAudiencia(transcricao), schemaAudiencia, 16000, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *claudeAnalyzer) ClassificarMovimentacao(ctx context.Context, descricao string, area domain.AreaDoDireito) (*ClassificacaoMovimentacao, error) {
	var out ClassificacaoMovimentacao
	// Classificação é uma tarefa curta: max_tokens baixo e esforço médio
	// bastam, e isso mantém a latência do webhook aceitável.
	if err := c.pedir(ctx, area, promptMovimentacao(descricao), schemaMovimentacao, 2000, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *claudeAnalyzer) pedir(ctx context.Context, area domain.AreaDoDireito, prompt string, schema map[string]any, maxTokens int64, dst any) error {
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.modelo),
		MaxTokens: maxTokens,
		// O prefixo estável (identidade + norma da área) fica em cache; só o
		// material analisado varia entre chamadas.
		System: []anthropic.TextBlockParam{{
			Text:         promptSistema(area),
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{},
		},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortHigh,
			Format: anthropic.JSONOutputFormatParam{Schema: schema},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return fmt.Errorf("claude: %w", err)
	}

	// Recusa por política vem como HTTP 200 — checar StopReason antes de ler
	// o conteúdo, senão o erro passa como resposta vazia.
	if resp.StopReason == anthropic.StopReasonRefusal {
		return fmt.Errorf("claude recusou a análise: %s", resp.StopDetails.Explanation)
	}

	var sb strings.Builder
	for _, bloco := range resp.Content {
		if t, ok := bloco.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(t.Text)
		}
	}
	bruto := strings.TrimSpace(sb.String())
	if bruto == "" {
		return fmt.Errorf("claude devolveu resposta vazia (stop_reason=%s)", resp.StopReason)
	}

	if err := json.Unmarshal([]byte(bruto), dst); err != nil {
		return fmt.Errorf("resposta do claude fora do esquema: %w", err)
	}
	return nil
}
