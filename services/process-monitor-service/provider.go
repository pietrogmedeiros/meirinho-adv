package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/pietromedeiros/meirinho/internal/platform/config"
)

// MovimentacaoProvider é uma movimentação como o tribunal a publica, antes de
// virar registro nosso.
//
// EventID é a chave de idempotência: o provider reenvia o mesmo andamento em
// consultas consecutivas, e é o índice único (process_id, provider_event_id)
// que garante que ele entre uma vez só.
type MovimentacaoProvider struct {
	EventID   string
	Descricao string
	Data      time.Time
}

// Provider é a fonte de andamentos processuais. A interface é estreita de
// propósito: trocar o mock por DataJud/CNJ, PJe ou um agregador comercial não
// deve tocar no poller nem no banco.
type Provider interface {
	Movimentacoes(ctx context.Context, numeroCNJ, ref string, desde time.Time) ([]MovimentacaoProvider, error)
	Nome() string
}

func escolherProvider() (Provider, error) {
	nome := config.String("PROCESS_PROVIDER", "mock")
	switch nome {
	case "mock":
		return &mockProvider{
			chance: config.Float("MOCK_MOVEMENT_CHANCE", 0.35),
		}, nil
	default:
		// Falha explícita no boot: um provider desconhecido que silenciosamente
		// virasse mock faria o advogado achar que está monitorando de verdade.
		return nil, fmt.Errorf("PROCESS_PROVIDER=%q não implementado (disponível: mock)", nome)
	}
}

// mockProvider simula o diário oficial para que o pipeline seja demonstrável
// sem credencial de tribunal.
//
// É determinístico por (CNJ, dia): duas consultas no mesmo dia devolvem a mesma
// movimentação com o mesmo EventID, então a idempotência do banco é exercitada
// de verdade em vez de ser mascarada por dado sempre novo.
type mockProvider struct{ chance float64 }

func (m *mockProvider) Nome() string { return "mock" }

var frasesMock = []string{
	"Intimação da parte autora para manifestar-se sobre a contestação, prazo de 15 dias.",
	"Juntada de petição de procuração.",
	"Despacho: designada audiência de conciliação.",
	"Publicado no Diário da Justiça Eletrônico.",
	"Decisão: deferida a produção de prova pericial.",
	"Certidão de decurso de prazo sem manifestação.",
	"Sentença publicada. Prazo recursal em curso.",
	"Autos distribuídos por sorteio à vara competente.",
}

func (m *mockProvider) Movimentacoes(ctx context.Context, numeroCNJ, ref string, desde time.Time) ([]MovimentacaoProvider, error) {
	dia := time.Now().UTC().Format("2006-01-02")
	semente := sha256.Sum256([]byte(numeroCNJ + "|" + dia))
	rng := rand.New(rand.NewSource(int64(binary.BigEndian.Uint64(semente[:8]))))

	if rng.Float64() > m.chance {
		return nil, nil // nada novo hoje, o caso comum
	}

	frase := frasesMock[rng.Intn(len(frasesMock))]
	return []MovimentacaoProvider{{
		EventID:   fmt.Sprintf("mock-%s-%s", dia, strings.ReplaceAll(numeroCNJ, ".", "")),
		Descricao: frase,
		Data:      time.Now().UTC(),
	}}, nil
}
