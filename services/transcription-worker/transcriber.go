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
)

// Transcriber isola o motor de transcrição do resto do pipeline.
//
// Existe para que o protótipo atual (Whisper rodando local) seja um detalhe
// substituível: trocar por outro modelo, por uma API ou por um serviço de GPU
// não toca em nenhuma outra parte do sistema.
type Transcriber interface {
	Transcrever(ctx context.Context, caminhoAudio string) (string, error)
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

func (w *whisperCLI) Transcrever(ctx context.Context, caminhoAudio string) (string, error) {
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

func (m *mockTranscriber) Transcrever(ctx context.Context, caminhoAudio string) (string, error) {
	// Atraso proposital: a tela de status acompanha o processamento em tempo
	// real, e sem isso o estado "transcrevendo" nunca seria visível.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(m.atraso):
	}

	info, err := os.Stat(caminhoAudio)
	tamanho := int64(0)
	if err == nil {
		tamanho = info.Size()
	}

	return fmt.Sprintf(`[TRANSCRIÇÃO SIMULADA — motor mock, arquivo de %d KB]

Juiz: Declaro aberta a audiência. Presentes as partes e seus procuradores.

Advogado do autor: Excelência, reitero os termos da inicial. A prova documental
juntada às fls. 45/62 demonstra o inadimplemento contratual desde março.

Advogado do réu: Impugno os documentos. Sustento que houve novação da dívida
por acordo verbal posterior, o que afasta a mora alegada.

Juiz: Defiro a oitiva da testemunha arrolada pelo réu. Designo audiência em
continuação. Intimem-se as partes. Prazo de 15 dias para memoriais.

[Fim da gravação]`, tamanho/1024), nil
}
