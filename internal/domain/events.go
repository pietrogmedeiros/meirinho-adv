package domain

import "time"

// Nomes dos streams Redis. O fluxo é:
//
//	audio.uploaded    -> transcription-worker  -> audio.transcribed
//	audio.transcribed -> analysis-service      -> hearing.analyzed
//	hearing.analyzed  -> notification-service
//
//	movement.received  -> classification-worker -> movement.classified
//	movement.classified-> notification-service
const (
	StreamAudioUploaded      = "audio.uploaded"
	StreamAudioTranscribed   = "audio.transcribed"
	StreamHearingAnalyzed    = "hearing.analyzed"
	StreamMovementReceived   = "movement.received"
	StreamMovementClassified = "movement.classified"
)

// Envelope é o formato de todo evento no barramento. TenantID viaja sempre
// junto: os workers usam ele para abrir a transação já escopada por tenant
// (RLS), então nenhum consumidor precisa "descobrir" de quem é o dado.
type Envelope struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	TenantID  string         `json:"tenant_id"`
	OccuredAt time.Time      `json:"occurred_at"`
	Data      map[string]any `json:"-"`
}

type AudioUploaded struct {
	HearingID string        `json:"hearing_id"`
	TenantID  string        `json:"tenant_id"`
	ObjectKey string        `json:"object_key"`
	Area      AreaDoDireito `json:"area_do_direito"`
}

type AudioTranscribed struct {
	HearingID  string        `json:"hearing_id"`
	TenantID   string        `json:"tenant_id"`
	Transcript string        `json:"transcript"`
	Area       AreaDoDireito `json:"area_do_direito"`
}

type HearingAnalyzed struct {
	HearingID  string `json:"hearing_id"`
	TenantID   string `json:"tenant_id"`
	Summary    string `json:"summary"`
	Suggestion string `json:"suggestion"`
}

type MovementReceived struct {
	MovementID string        `json:"movement_id"`
	ProcessID  string        `json:"process_id"`
	TenantID   string        `json:"tenant_id"`
	NumeroCNJ  string        `json:"numero_cnj"`
	Descricao  string        `json:"descricao"`
	Data       time.Time     `json:"data"`
	Area       AreaDoDireito `json:"area_do_direito"`
}

type MovementClassified struct {
	MovementID       string   `json:"movement_id"`
	ProcessID        string   `json:"process_id"`
	TenantID         string   `json:"tenant_id"`
	NumeroCNJ        string   `json:"numero_cnj"`
	Descricao        string   `json:"descricao"`
	Urgencia         Urgencia `json:"urgencia"`
	Sugestao         string   `json:"sugestao"`
	PrazoDias        *int     `json:"prazo_dias,omitempty"`
	FallbackAplicado bool     `json:"fallback_aplicado"`
}
