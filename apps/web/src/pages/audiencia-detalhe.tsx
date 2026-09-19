import { useState } from "react"
import { Link, useParams } from "react-router-dom"
import { AlertTriangle, ArrowLeft, ChevronDown, FileText, Lightbulb, ListChecks, Loader2 } from "lucide-react"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { MensagemErro } from "@/components/estados"
import { api } from "@/lib/api"
import { formatarData, rotuloArea, STATUS_AUDIENCIA, urlDeAudio } from "@/lib/format"
import { useDados, usePolling } from "@/lib/hooks"
import type { HearingStatus } from "@/lib/tipos"

// Ordem em que o pipeline percorre os status — usada para mostrar em que
// etapa a audiência está enquanto o resultado não chega.
const ETAPAS: { status: HearingStatus[]; rotulo: string }[] = [
  { status: ["uploaded"], rotulo: "Recebido" },
  { status: ["transcribing"], rotulo: "Transcrevendo" },
  { status: ["transcribed", "analyzing"], rotulo: "Analisando" },
  { status: ["analyzed"], rotulo: "Pronta" },
]

export function AudienciaDetalhePage() {
  const { id = "" } = useParams()
  const { dado: aud, erro, carregando, recarregar } = useDados(() => api.audiencia(id), [id])

  const st = aud ? (STATUS_AUDIENCIA[aud.status] ?? STATUS_AUDIENCIA.uploaded) : null
  usePolling(() => void recarregar(true), !!st?.emAndamento)

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" className="-ml-2 gap-1.5" render={<Link to="/audiencias" />}>
        <ArrowLeft className="size-4" />
        Audiências
      </Button>

      {erro && <MensagemErro texto={erro} aoTentarNovamente={() => void recarregar()} />}

      {carregando ? (
        <div className="space-y-4">
          <Skeleton className="h-10 w-2/3" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : aud && st ? (
        <>
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="min-w-0 space-y-1">
              <h1 className="truncate text-2xl font-semibold tracking-tight">
                {aud.titulo || aud.nome_arquivo}
              </h1>
              <p className="text-sm text-muted-foreground">
                {rotuloArea(aud.area_do_direito)} · enviada em {formatarData(aud.created_at)}
              </p>
            </div>
            <Badge variant={st.variante} className="gap-1.5">
              {st.emAndamento && <Loader2 className="size-3 animate-spin" />}
              {st.rotulo}
            </Badge>
          </div>

          {aud.audio_url && (
            <audio controls preload="metadata" src={urlDeAudio(aud.audio_url)} className="w-full" />
          )}

          {st.emAndamento && <Progresso status={aud.status} />}

          {aud.status === "failed" && (
            <Alert variant="destructive">
              <AlertTriangle className="size-4" />
              <AlertTitle>O processamento falhou</AlertTitle>
              <AlertDescription>
                {aud.erro || "Não foi possível transcrever ou analisar este áudio."}
              </AlertDescription>
            </Alert>
          )}

          {aud.summary && (
            <Secao Icone={FileText} titulo="Resumo">
              <p className="whitespace-pre-line leading-relaxed">{aud.summary}</p>
            </Secao>
          )}

          {aud.suggestion && (
            <Secao Icone={Lightbulb} titulo="Sugestão estratégica">
              <p className="whitespace-pre-line leading-relaxed">{aud.suggestion}</p>
            </Secao>
          )}

          {!!aud.pontos_criticos?.length && (
            <Secao Icone={ListChecks} titulo="Pontos críticos">
              <ul className="space-y-2">
                {aud.pontos_criticos.map((p, i) => (
                  <li key={i} className="flex gap-3">
                    <span className="mt-2 size-1.5 shrink-0 rounded-full bg-amber-500" />
                    <span className="leading-relaxed">{p}</span>
                  </li>
                ))}
              </ul>
            </Secao>
          )}

          {aud.transcript && <Transcricao texto={aud.transcript} />}
        </>
      ) : null}
    </div>
  )
}

function Progresso({ status }: { status: HearingStatus }) {
  const atual = ETAPAS.findIndex((e) => e.status.includes(status))
  return (
    <Card className="py-0">
      <CardContent className="space-y-3 p-4">
        <div className="grid grid-cols-4 gap-2">
          {ETAPAS.map((e, i) => (
            <div key={e.rotulo} className="space-y-1.5">
              <div className={i <= atual ? "h-1.5 rounded-full bg-primary" : "h-1.5 rounded-full bg-muted"} />
              <p className={i <= atual ? "text-xs font-medium" : "text-xs text-muted-foreground"}>
                {e.rotulo}
              </p>
            </div>
          ))}
        </div>
        <p className="text-sm text-muted-foreground">
          Pode sair desta página: o processamento continua e você recebe uma notificação ao final.
        </p>
      </CardContent>
    </Card>
  )
}

function Secao({
  Icone, titulo, children,
}: {
  Icone: React.ComponentType<{ className?: string }>
  titulo: string
  children: React.ReactNode
}) {
  return (
    <Card>
      <CardHeader className="pb-0">
        <CardTitle className="flex items-center gap-2 text-base">
          <Icone className="size-4 text-muted-foreground" />
          {titulo}
        </CardTitle>
      </CardHeader>
      <CardContent className="text-sm">{children}</CardContent>
    </Card>
  )
}

function Transcricao({ texto }: { texto: string }) {
  // Fechada por padrão: é o material bruto, longo, e só interessa quando o
  // advogado quer conferir de onde veio um ponto do resumo.
  const [aberta, setAberta] = useState(false)
  return (
    <Card>
      <CardHeader className="pb-0">
        <button
          type="button"
          onClick={() => setAberta((a) => !a)}
          className="flex w-full items-center justify-between text-left"
          aria-expanded={aberta}
        >
          <CardTitle className="text-base">Transcrição completa</CardTitle>
          <ChevronDown className={aberta ? "size-4 rotate-180 transition-transform" : "size-4 transition-transform"} />
        </button>
      </CardHeader>
      {aberta && (
        <CardContent>
          <p className="max-h-[32rem] overflow-y-auto whitespace-pre-line text-sm leading-relaxed text-muted-foreground">
            {texto}
          </p>
        </CardContent>
      )}
    </Card>
  )
}
