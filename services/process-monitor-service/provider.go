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

// etapasMock seguem a ordem de um processo de conhecimento. O histórico de
// cada processo é um prefixo desta lista, então nunca aparece sentença antes
// da citação.
var etapasMock = []string{
	"Autos distribuídos por sorteio à vara competente.",
	"Despacho: cite-se o réu para apresentar contestação no prazo legal.",
	"Juntada de mandado de citação cumprido.",
	"Juntada de contestação com documentos.",
	"Intimação da parte autora para réplica à contestação, prazo de 15 dias.",
	"Juntada de petição de réplica.",
	"Decisão saneadora: fixados os pontos controvertidos e deferida a produção de prova pericial.",
	"Juntada de laudo pericial.",
	"Despacho: designada audiência de instrução e julgamento.",
	"Conclusos para sentença.",
	"Sentença publicada. Prazo recursal em curso.",
}

// frasesMock são andamentos avulsos que podem surgir a qualquer momento — o
// "novo hoje" que o poller descobre depois do histórico.
var frasesMock = []string{
	"Intimação da parte autora para manifestar-se sobre documentos novos, prazo de 5 dias.",
	"Juntada de petição de procuração.",
	"Publicado no Diário da Justiça Eletrônico.",
	"Embargos de declaração opostos pela parte contrária.",
	"Certidão de decurso de prazo sem manifestação.",
	"Remetidos os autos ao contador judicial.",
}

func (m *mockProvider) Movimentacoes(ctx context.Context, numeroCNJ, ref string, desde time.Time) ([]MovimentacaoProvider, error) {
	cnj := strings.ReplaceAll(numeroCNJ, ".", "")
	agora := time.Now().UTC()
	movs := m.historico(cnj, agora)

	dia := agora.Format("2006-01-02")
	semente := sha256.Sum256([]byte(numeroCNJ + "|" + dia))
	rng := rand.New(rand.NewSource(int64(binary.BigEndian.Uint64(semente[:8]))))
	if rng.Float64() <= m.chance {
		movs = append(movs, MovimentacaoProvider{
			EventID:   fmt.Sprintf("mock-%s-%s", dia, cnj),
			Descricao: frasesMock[rng.Intn(len(frasesMock))],
			Data:      agora,
		})
	}
	return movs, nil
}

// historico devolve o passado do processo, como o tribunal devolveria na
// primeira consulta de um CNJ que já tramita. Depende só do CNJ: o EventID é
// estável e as consultas seguintes caem no ON CONFLICT DO NOTHING. As datas são
// relativas a agora, mas só a primeira gravação conta.
func (m *mockProvider) historico(cnj string, agora time.Time) []MovimentacaoProvider {
	semente := sha256.Sum256([]byte(cnj + "|historico"))
	rng := rand.New(rand.NewSource(int64(binary.BigEndian.Uint64(semente[:8]))))

	n := 3 + rng.Intn(len(etapasMock)-2) // de 3 até todas as etapas
	// Espaça as etapas de trás para frente: a última é recente (0 a 6 dias),
	// e cada anterior fica de 8 a 40 dias antes da seguinte.
	datas := make([]time.Time, n)
	d := agora.Add(-time.Duration(rng.Intn(7)*24+rng.Intn(10)) * time.Hour)
	for i := n - 1; i >= 0; i-- {
		datas[i] = d
		d = d.Add(-time.Duration(8+rng.Intn(33)) * 24 * time.Hour)
	}

	movs := make([]MovimentacaoProvider, n)
	for i := 0; i < n; i++ {
		movs[i] = MovimentacaoProvider{
			EventID:   fmt.Sprintf("mock-hist-%02d-%s", i, cnj),
			Descricao: etapasMock[i],
			Data:      datas[i],
		}
	}
	return movs
}
