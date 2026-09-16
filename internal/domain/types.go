// Package domain concentra os tipos compartilhados entre os serviços do Meirinho.
// Qualquer coisa que atravesse a fronteira de um serviço (evento, enum de status,
// contrato de payload) mora aqui — o resto fica privado no serviço.
package domain

import "strings"

// AreaDoDireito é a norma contra a qual a audiência ou a movimentação é analisada.
type AreaDoDireito string

const (
	AreaCivel          AreaDoDireito = "civel"
	AreaFamilia        AreaDoDireito = "familia"
	AreaCriminal       AreaDoDireito = "criminal"
	AreaTrabalhista    AreaDoDireito = "trabalhista"
	AreaAdministrativo AreaDoDireito = "administrativo"
)

var areasValidas = map[AreaDoDireito]string{
	AreaCivel:          "Direito Civil e Processo Civil",
	AreaFamilia:        "Direito de Família e Sucessões",
	AreaCriminal:       "Direito Penal e Processo Penal",
	AreaTrabalhista:    "Direito do Trabalho e Processo do Trabalho",
	AreaAdministrativo: "Direito Administrativo",
}

func (a AreaDoDireito) Valida() bool {
	_, ok := areasValidas[a]
	return ok
}

// Norma devolve a descrição da área usada para ancorar o prompt de análise.
func (a AreaDoDireito) Norma() string {
	if d, ok := areasValidas[a]; ok {
		return d
	}
	return areasValidas[AreaCivel]
}

func ParseArea(s string) (AreaDoDireito, bool) {
	a := AreaDoDireito(strings.ToLower(strings.TrimSpace(s)))
	return a, a.Valida()
}

// HearingStatus acompanha o pipeline de uma audiência.
type HearingStatus string

const (
	StatusUploaded     HearingStatus = "uploaded"
	StatusTranscribing HearingStatus = "transcribing"
	StatusTranscribed  HearingStatus = "transcribed"
	StatusAnalyzing    HearingStatus = "analyzing"
	StatusAnalyzed     HearingStatus = "analyzed"
	StatusFailed       HearingStatus = "failed"
)

// Urgencia é o resultado da classificação de uma movimentação processual.
// A ordem importa: Nivel() é usado pela regra de fallback conservador, que
// nunca pode rebaixar uma classificação.
type Urgencia string

const (
	UrgenciaAlta    Urgencia = "alta"
	UrgenciaMedia   Urgencia = "media"
	UrgenciaBaixa   Urgencia = "baixa"
	UrgenciaNenhuma Urgencia = "nenhuma"
)

func (u Urgencia) Nivel() int {
	switch u {
	case UrgenciaAlta:
		return 3
	case UrgenciaMedia:
		return 2
	case UrgenciaBaixa:
		return 1
	case UrgenciaNenhuma:
		return 0
	}
	return -1 // desconhecida
}

func (u Urgencia) Valida() bool { return u.Nivel() >= 0 }

// TipoNotificacao — whatsapp existe no modelo desde já, mas o envio fica atrás
// de feature flag desligada no MVP.
type TipoNotificacao string

const (
	NotifInApp    TipoNotificacao = "in_app"
	NotifEmail    TipoNotificacao = "email"
	NotifWhatsApp TipoNotificacao = "whatsapp"
)
