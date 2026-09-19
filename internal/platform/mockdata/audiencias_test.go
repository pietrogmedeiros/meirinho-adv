package mockdata

import "testing"

// Cada caso precisa ser alcançável pelo título e voltar da transcrição para a
// própria análise; senão a demo mostra resumo de uma audiência e transcrição
// de outra.
func TestCasosCoerentes(t *testing.T) {
	vistos := map[string]bool{}
	for _, c := range Casos {
		if vistos[c.Transcricao] {
			t.Errorf("transcrição duplicada no caso %q", c.Chave)
		}
		vistos[c.Transcricao] = true

		if got := Escolher(c.Area, "Audiência — "+c.Chave); got.Chave != c.Chave {
			t.Errorf("título com chave %q escolheu o caso %q", c.Chave, got.Chave)
		}
		if got := PorTranscricao(c.Transcricao, c.Area); got.Chave != c.Chave {
			t.Errorf("transcrição do caso %q voltou como %q", c.Chave, got.Chave)
		}
		if c.Resumo == "" || c.Sugestao == "" || len(c.Pontos) == 0 {
			t.Errorf("caso %q incompleto", c.Chave)
		}
	}
}

func TestEscolherSemChaveFicaNaArea(t *testing.T) {
	for _, c := range Casos {
		if got := Escolher(c.Area, "gravação sem nome conhecido"); got.Area != c.Area {
			t.Errorf("área %s escolheu caso da área %s", c.Area, got.Area)
		}
	}
}
