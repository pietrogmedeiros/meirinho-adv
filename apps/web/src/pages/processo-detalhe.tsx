import { useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { Archive, ArrowLeft, CalendarClock, HelpCircle, Inbox, Lightbulb, Loader2 } from "lucide-react"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger,
} from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { EstadoVazio, MensagemErro } from "@/components/estados"
import { SeloUrgencia } from "@/components/urgencia"
import { api } from "@/lib/api"
import { formatarCNJ, formatarData, formatarDataCurta, rotuloArea, URGENCIA } from "@/lib/format"
import { useDados, usePolling } from "@/lib/hooks"
import type { Movimentacao, Processo } from "@/lib/tipos"
import { cn } from "@/lib/utils"

export function ProcessoDetalhePage() {
  const { id = "" } = useParams()

  const processo = useDados(() => api.processo(id), [id])
  const movs = useDados(() => api.movimentacoes(id), [id])

  // Movimentação nova chega do poller e é classificada por um worker; sem
  // polling, a urgência de uma recém-chegada ficaria "classificando" até F5.
  // Enquanto houver alguma sem classificação, o intervalo é curto.
  const ativo = processo.dado?.status !== "arquivado"
  const classificando = (movs.dado ?? []).some((m) => !m.urgencia)
  const vazio = movs.dado?.length === 0
  usePolling(() => void movs.recarregar(true), ativo, classificando || vazio ? 3000 : 15000)

  const erro = processo.erro ?? movs.erro

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" className="-ml-2 gap-1.5" render={<Link to="/processos" />}>
        <ArrowLeft className="size-4" />
        Processos
      </Button>

      {erro && (
        <MensagemErro
          texto={erro}
          aoTentarNovamente={() => {
            void processo.recarregar()
            void movs.recarregar()
          }}
        />
      )}

      {processo.carregando ? (
        <Skeleton className="h-16 w-2/3" />
      ) : processo.dado ? (
        <Cabecalho p={processo.dado} />
      ) : null}

      {processo.dado && (
        <section className="space-y-3">
          <h2 className="text-sm font-medium text-muted-foreground">Movimentações</h2>
          {movs.carregando ? (
            <div className="space-y-3">
              {[0, 1, 2].map((i) => <Skeleton key={i} className="h-24 w-full" />)}
            </div>
          ) : (movs.dado ?? []).length === 0 && !movs.erro ? (
            <EstadoVazio
              Icone={Inbox}
              titulo="Nenhuma movimentação ainda"
              descricao="O tribunal é consultado periodicamente; os andamentos aparecem aqui já classificados por urgência."
            />
          ) : (
            <ol className="relative space-y-3 before:absolute before:top-2 before:bottom-2 before:left-[7px] before:w-px before:bg-border">
              {(movs.dado ?? []).map((m) => <ItemMovimentacao key={m.id} m={m} />)}
            </ol>
          )}
        </section>
      )}
    </div>
  )
}

function Cabecalho({ p }: { p: Processo }) {
  const arquivado = p.status === "arquivado"
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="min-w-0 space-y-1">
        <div className="flex items-center gap-2">
          <h1 className="truncate text-2xl font-semibold tracking-tight">
            {p.titulo || formatarCNJ(p.numero_cnj)}
          </h1>
          {arquivado && <Badge variant="outline">Arquivado</Badge>}
        </div>
        <p className="text-sm text-muted-foreground">
          <span className="font-mono">{formatarCNJ(p.numero_cnj)}</span>
          {p.tribunal && ` · ${p.tribunal}`}
          {` · ${rotuloArea(p.area_do_direito)}`}
          {` · monitorado desde ${formatarDataCurta(p.created_at)}`}
          {` · ${p.total_movimentacoes} ${p.total_movimentacoes === 1 ? "movimentação" : "movimentações"}`}
        </p>
      </div>
      {!arquivado && <DialogArquivar p={p} />}
    </div>
  )
}

function ItemMovimentacao({ m }: { m: Movimentacao }) {
  const urg = m.urgencia ? URGENCIA[m.urgencia] : null
  return (
    <li className="relative pl-7">
      <span
        className={cn(
          "absolute top-4 left-0 size-[15px] rounded-full border-[3px] border-background",
          urg ? urg.cor : "bg-muted-foreground/40",
        )}
      />
      <Card className={cn("py-0", m.urgencia === "alta" && "border-red-500/40")}>
        <CardContent className="space-y-3 p-4">
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className="font-medium">{formatarData(m.data)}</span>
            <span className="flex-1" />
            {m.urgencia ? (
              <SeloUrgencia urgencia={m.urgencia} prefixo />
            ) : (
              <Badge variant="secondary" className="gap-1.5">
                <Loader2 className="size-3 animate-spin" />
                Classificando
              </Badge>
            )}
            {m.prazo_dias != null && m.prazo_dias > 0 && (
              <Badge variant="outline" className="gap-1.5">
                <CalendarClock className="size-3" />
                Prazo de {m.prazo_dias} {m.prazo_dias === 1 ? "dia" : "dias"}
              </Badge>
            )}
          </div>

          <p className="whitespace-pre-line text-sm leading-relaxed">{m.descricao}</p>

          {m.fallback_aplicado && (
            <p className="flex items-start gap-2 rounded-md bg-amber-500/10 p-2.5 text-sm text-amber-800 dark:text-amber-300">
              <HelpCircle className="mt-0.5 size-4 shrink-0" />
              A classificação automática ficou incerta; a urgência foi elevada por precaução. Leia o andamento.
            </p>
          )}

          {m.sugestao && (
            <p className="flex items-start gap-2 text-sm text-muted-foreground">
              <Lightbulb className="mt-0.5 size-4 shrink-0" />
              {m.sugestao}
            </p>
          )}
        </CardContent>
      </Card>
    </li>
  )
}

function DialogArquivar({ p }: { p: Processo }) {
  const navegar = useNavigate()
  const [enviando, setEnviando] = useState(false)

  async function arquivar() {
    setEnviando(true)
    try {
      await api.arquivarProcesso(p.id)
      toast.success("Processo arquivado", { description: "O monitoramento foi interrompido." })
      navegar("/processos")
    } catch (err) {
      toast.error("Não foi possível arquivar", {
        description: err instanceof Error ? err.message : undefined,
      })
      setEnviando(false)
    }
  }

  return (
    <Dialog>
      <DialogTrigger render={<Button variant="outline" className="gap-2" />}>
        <Archive className="size-4" />
        Arquivar
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Arquivar processo?</DialogTitle>
          <DialogDescription>
            O Meirinho deixa de consultar o tribunal e não haverá novos alertas para{" "}
            <span className="font-mono">{formatarCNJ(p.numero_cnj)}</span>. As movimentações já
            recebidas continuam disponíveis.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>Cancelar</DialogClose>
          <Button variant="destructive" onClick={arquivar} disabled={enviando} className="gap-2">
            {enviando && <Loader2 className="size-4 animate-spin" />}
            Arquivar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
