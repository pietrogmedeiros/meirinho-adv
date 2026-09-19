// Espelho dos contratos do backend. Mantido à mão e em português porque é isso
// que a API devolve — traduzir os campos aqui só criaria um dicionário a mais
// para manter sincronizado.

export type AreaDoDireito =
  | "civel"
  | "familia"
  | "criminal"
  | "trabalhista"
  | "administrativo"

export const AREAS: { valor: AreaDoDireito; rotulo: string }[] = [
  { valor: "civel", rotulo: "Cível" },
  { valor: "familia", rotulo: "Família e Sucessões" },
  { valor: "criminal", rotulo: "Criminal" },
  { valor: "trabalhista", rotulo: "Trabalhista" },
  { valor: "administrativo", rotulo: "Administrativo" },
]

export type HearingStatus =
  | "uploaded"
  | "transcribing"
  | "transcribed"
  | "analyzing"
  | "analyzed"
  | "failed"

export type Urgencia = "alta" | "media" | "baixa" | "nenhuma"

export interface Tenant {
  id: string
  nome: string
  oab: string
  email: string
  retencao_dias?: number
}

export interface Sessao {
  token: string
  expira_em: string
  tenant: Tenant
}

export interface Audiencia {
  id: string
  titulo: string
  nome_arquivo: string
  area_do_direito: AreaDoDireito
  status: HearingStatus
  transcript?: string
  summary?: string
  suggestion?: string
  pontos_criticos?: string[]
  erro?: string
  audio_url?: string
  created_at: string
  updated_at: string
}

export interface Processo {
  id: string
  numero_cnj: string
  titulo: string
  tribunal: string
  area_do_direito: AreaDoDireito
  provider: string
  status: "ativo" | "arquivado"
  created_at: string
  total_movimentacoes: number
  ultima_movimentacao?: string
  ultima_descricao?: string
  ultima_urgencia?: Urgencia
}

export interface Movimentacao {
  id: string
  process_id: string
  descricao: string
  data: string
  urgencia?: Urgencia
  sugestao?: string
  prazo_dias?: number
  fallback_aplicado: boolean
  created_at: string
}

export interface PreferenciasNotificacao {
  urgencia_minima: Urgencia
  alertar_audiencias: boolean
  canal_email: boolean
  canal_whatsapp: boolean
  telefone: string
  canais_disponiveis?: { email: boolean; whatsapp: boolean }
}

export interface Notificacao {
  id: string
  tipo: "in_app" | "email" | "whatsapp"
  categoria: string
  titulo: string
  corpo: string
  urgencia?: Urgencia
  referencia_id?: string
  lida: boolean
  created_at: string
}
