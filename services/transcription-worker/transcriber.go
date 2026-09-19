package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pietromedeiros/meirinho/internal/domain"
	"github.com/pietromedeiros/meirinho/internal/platform/mockdata"
)

// Transcriber isola o motor de transcrição do resto do pipeline.
//
// Existe para que o protótipo atual (Whisper rodando local) seja um detalhe
// substituível: trocar por outro modelo, por uma API ou por um serviço de GPU
// não toca em nenhuma outra parte do sistema.
type Transcriber interface {
	// area e titulo não mudam o motor real (o Whisper transcreve o que ouve);
	// o mock os usa para escolher uma audiência fictícia coerente.
	Transcrever(ctx context.Context, caminhoAudio string, area domain.AreaDoDireito, titulo string) (string, error)
	Nome() string
}

// whisperCLI executa o binário do Whisper (ou faster-whisper) instalado na
// imagem do worker.
type whisperCLI struct {
	binario string
	modelo  string
	idioma  string
	timeout time.Duration
}

func (w *whisperCLI) Nome() string { return "whisper-cli" }

func (w *whisperCLI) Transcrever(ctx context.Context, caminhoAudio string, _ domain.AreaDoDireito, _ string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	saida, err := os.MkdirTemp("", "whisper-*")
	if err != nil {
		return "", fmt.Errorf("criar diretório de saída: %w", err)
	}
	defer os.RemoveAll(saida)

	cmd := exec.CommandContext(ctx, w.binario,
		caminhoAudio,
		"--model", w.modelo,
		"--language", w.idioma,
		"--output_format", "txt",
		"--output_dir", saida,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper falhou: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	base := strings.TrimSuffix(filepath.Base(caminhoAudio), filepath.Ext(caminhoAudio))
	txt, err := os.ReadFile(filepath.Join(saida, base+".txt"))
	if err != nil {
		return "", fmt.Errorf("ler transcrição: %w", err)
	}

	transcricao := strings.TrimSpace(string(txt))
	if transcricao == "" {
		return "", fmt.Errorf("whisper devolveu transcrição vazia")
	}
	return transcricao, nil
}

// mockTranscriber permite rodar o fluxo ponta a ponta sem o modelo instalado —
// é o que mantém a demo do MVP funcionando em qualquer máquina.
type mockTranscriber struct{ atraso time.Duration }

func (m *mockTranscriber) Nome() string { return "mock" }

func (m *mockTranscriber) Transcrever(ctx context.Context, caminhoAudio string, area domain.AreaDoDireito, titulo string) (string, error) {
	// Atraso proposital: a tela de status acompanha o processamento em tempo
	// real, e sem isso o estado "transcrevendo" nunca seria visível.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(m.atraso):
	}

	return mockdata.Escolher(area, titulo).Transcricao, nil
}
