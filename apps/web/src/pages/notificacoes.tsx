import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Bell, CheckCheck, FileAudio, Scroll } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { EstadoVazio, MensagemErro } from "@/components/estados"
import { SeloUrgencia } from "@/components/urgencia"
import { api } from "@/lib/api"
import { formatarData } from "@/lib/format"
import { useDados, usePolling } from "@/lib/hooks"
import { rotaDaNotificacao } from "@/lib/notificacoes"
import type { Notificacao } from "@/lib/tipos"
import { cn } from "@/lib/utils"

export function NotificacoesPage() {
  const navegar = useNavigate()
  const [filtro, setFiltro] = useState<"todas" | "nao_lidas">("todas")
  const { dado, erro, carregando, recarregar } = useDados(
    () => api.notificacoes(filtro === "nao_lidas"),
    [filtro],
  )
  usePolling(() => void recarregar(true), true, 15000)

  const itens = dado ?? []
  const haNaoLidas = itens.some((n) => !n.lida)

  async function abrir(n: Notificacao) {
    if (!n.lida) {
      // Marca antes de navegar, sem bloquear: se falhar, o pior caso é a
      // notificação continuar em negrito na próxima visita.
      void api.marcarLida(n.id).catch(() => {})
    }
    const destino = await rotaDaNotificacao(n)
    if (destino) navegar(destino)
    else void recarregar(true)
  }

  async function marcarTodas() {
    try {
      const { marcadas } = await api.marcarTodasLidas()
      toast.success(marcadas === 1 ? "1 notificação marcada como lida" : `${marcadas} notificações marcadas como lidas`)
      void recarregar(true)
    } catch (err) {
      toast.error("Não foi possível marcar", {
        description: err instanceof Error ? err.message : undefined,
      })
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Notificações</h1>
          <p className="text-sm text-muted-foreground">
            Análises concluídas e movimentações que pedem atenção.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Tabs value={filtro} onValueChange={(v) => setFiltro(v as typeof filtro)}>
            <TabsList>
              <TabsTrigger value="todas">Todas</TabsTrigger>
              <TabsTrigger value="nao_lidas">Não lidas</TabsTrigger>
            </TabsList>
          </Tabs>
          <Button variant="outline" size="sm" className="gap-2" onClick={marcarTodas} disabled={!haNaoLidas}>
            <CheckCheck className="size-4" />
            Marcar todas
          </Button>
        </div>
      </div>

      {erro && <MensagemErro texto={erro} aoTentarNovamente={() => void recarregar()} />}

      {carregando ? (
        <div className="space-y-3">
          {[0, 1, 2].map((i) => <Skeleton key={i} className="h-20 w-full" />)}
        </div>
      ) : itens.length === 0 && !erro ? (
        <EstadoVazio
          Icone={Bell}
          titulo={filtro === "nao_lidas" ? "Tudo lido" : "Nenhuma notificação"}
          descricao={
            filtro === "nao_lidas"
              ? "Não há nada pendente de leitura."
              : "Você será avisado aqui quando uma análise terminar ou um processo tiver movimentação relevante."
          }
        />
      ) : (
        <div className="space-y-2">
          {itens.map((n) => (
            <ItemNotificacao key={n.id} n={n} aoAbrir={() => void abrir(n)} />
          ))}
        </div>
      )}
    </div>
  )
}

function ItemNotificacao({ n, aoAbrir }: { n: Notificacao; aoAbrir: () => void }) {
  const Icone = n.categoria === "audiencia" ? FileAudio : Scroll
  return (
    <button type="button" onClick={aoAbrir} className="block w-full text-left">
      <Card className={cn("py-0 transition-colors hover:border-primary/40", n.lida && "bg-background/60")}>
        <CardContent className="flex gap-4 p-4">
          <div className="relative flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted">
            <Icone className="size-5 text-muted-foreground" />
            {!n.lida && (
              <span className="absolute -top-0.5 -right-0.5 size-2.5 rounded-full bg-primary ring-2 ring-background" />
            )}
          </div>
          <div className="min-w-0 flex-1 space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <p className={cn("truncate", n.lida ? "text-muted-foreground" : "font-medium")}>{n.titulo}</p>
              {n.urgencia && n.urgencia !== "nenhuma" && <SeloUrgencia urgencia={n.urgencia} />}
            </div>
            <p className="line-clamp-2 text-sm text-muted-foreground">{n.corpo}</p>
            <p className="text-xs text-muted-foreground">{formatarData(n.created_at)}</p>
          </div>
        </CardContent>
      </Card>
    </button>
  )
}
