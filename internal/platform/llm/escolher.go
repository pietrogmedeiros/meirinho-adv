package llm

import (
	"log/slog"
	"time"

	"github.com/pietromedeiros/meirinho/internal/platform/config"
)

// Escolher decide entre o analyzer real e o mock, do mesmo jeito que o
// transcription-worker escolhe o motor de transcrição: o mock é o padrão para
// que o compose suba em qualquer máquina, e ANALYZER=claude liga o real.
//
// Se pedirem claude sem chave, cai no mock com aviso em vez de derrubar o
// serviço — um worker morto em loop de restart é pior que um worker degradado
// que diz no log o que está faltando.
func Escolher(log *slog.Logger) Analyzer {
	if config.String("ANALYZER", "mock") != "claude" {
		return &mockAnalyzer{atraso: config.Duration("MOCK_ANALYZE_DELAY", 2*time.Second)}
	}
	chave := config.String("ANTHROPIC_API_KEY", "")
	if chave == "" {
		log.Warn("ANALYZER=claude sem ANTHROPIC_API_KEY — usando mock")
		return &mockAnalyzer{atraso: config.Duration("MOCK_ANALYZE_DELAY", 2*time.Second)}
	}
	return NovoClaude(chave, config.String("ANTHROPIC_MODEL", "claude-sonnet-5"))
}
