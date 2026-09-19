import { AREAS, type HearingStatus, type Urgencia } from "./tipos"

/** O banco guarda só os 20 dígitos; ninguém lê processo sem pontuação. */
export function formatarCNJ(cnj: string): string {
  const d = cnj.replace(/\D/g, "")
  if (d.length !== 20) return cnj
  return `${d.slice(0, 7)}-${d.slice(7, 9)}.${d.slice(9, 13)}.${d.slice(13, 14)}.${d.slice(14, 16)}.${d.slice(16, 20)}`
}

export function formatarData(iso: string): string {
  return new Date(iso).toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function formatarDataCurta(iso: string): string {
  return new Date(iso).toLocaleDateString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  })
}

type Variante = "default" | "secondary" | "destructive" | "outline"

export const STATUS_AUDIENCIA: Record<
  HearingStatus,
  { rotulo: string; variante: Variante; emAndamento: boolean }
> = {
  uploaded: { rotulo: "Na fila", variante: "secondary", emAndamento: true },
  transcribing: { rotulo: "Transcrevendo", variante: "secondary", emAndamento: true },
  transcribed: { rotulo: "Transcrita", variante: "secondary", emAndamento: true },
  analyzing: { rotulo: "Analisando", variante: "secondary", emAndamento: true },
  analyzed: { rotulo: "Pronta", variante: "default", emAndamento: false },
  failed: { rotulo: "Falhou", variante: "destructive", emAndamento: false },
}

export const URGENCIA: Record<Urgencia, { rotulo: string; variante: Variante; cor: string }> = {
  alta: { rotulo: "Alta", variante: "destructive", cor: "bg-red-500" },
  media: { rotulo: "Média", variante: "default", cor: "bg-amber-500" },
  baixa: { rotulo: "Baixa", variante: "secondary", cor: "bg-sky-500" },
  nenhuma: { rotulo: "Nenhuma", variante: "outline", cor: "bg-muted-foreground" },
}

export function rotuloArea(area: string): string {
  return AREAS.find((a) => a.valor === area)?.rotulo ?? area
}

/**
 * A URL assinada sai do hearing-service com o host interno do MinIO
 * (`minio:9000`), que o navegador não resolve. O gateway expõe o bucket na
 * mesma origem e repassa o Host original, então basta trocar a origem — a
 * assinatura cobre caminho, query e Host, e os três chegam intactos ao MinIO.
 */
export function urlDeAudio(assinada: string): string {
  try {
    const u = new URL(assinada)
    return u.pathname + u.search
  } catch {
    return assinada
  }
}
