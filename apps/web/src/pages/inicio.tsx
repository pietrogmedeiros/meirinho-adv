import { Link } from "react-router-dom"
import { AlertTriangle, ArrowRight, Bell, CheckCircle2, FileAudio, Loader2, Scroll } from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { MensagemErro } from "@/components/estados"
import { SeloUrgencia } from "@/components/urgencia"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { formatarCNJ, formatarData, formatarDataCurta, rotuloArea, STATUS_AUDIENCIA } from "@/lib/format"
import { useDados, usePolling } from "@/lib/hooks"
import { cn } from "@/lib/utils"

function saudacao(): string {
  const h = new Date().getHours()
  if (h < 12) return "Bom dia"
  if (h < 18) return "Boa tarde"
  return "Boa noite"
}

export function InicioPage() {
  const { tenant } = useAuth()
  const { dado, erro, carregando, recarregar } = useDados(
    () =>
      Promise.all([api.audiencias(), api.processos(), api.notificacoes(true)]).then(
        ([audiencias, processos, naoLidas]) => ({ audiencias, processos, naoLidas }),
      ),
    [],
  )
  usePolling(() => void recarregar(true), true, 15000)

  const audiencias = dado?.audiencias ?? []
  const processos = dado?.processos ?? []
  const naoLidas = dado?.naoLidas ?? []

  const ativos = processos.filter((p) => p.status === "ativo")
  const urgentes = naoLidas.filter((n) => n.urgencia === "alta")
  const prontas = audiencias.filter((a) => a.status === "analyzed").length
  const processando = audiencias.filter((a) => STATUS_AUDIENCIA[a.status]?.emAndamento).length
  const recentes = [...ativos]
    .filter((p) => p.ultima_movimentacao)
    .sort((a, b) => (b.ultima_movimentacao ?? "").localeCompare(a.ultima_movimentacao ?? ""))
    .slice(0, 5)

  const data = new Date().toLocaleDateString("pt-BR", { weekday: "long", day: "numeric", month: "long" })
  const hoje = data.charAt(0).toUpperCase() + data.slice(1)

  return (
    <div className="space-y-6">
      <div>
        <p className="text-sm text-muted-foreground">{hoje}</p>
        <h1 className="text-2xl font-semibold tracking-tight">
          {saudacao()}, {tenant?.nome?.split(" ").slice(0, 2).join(" ")}
        </h1>
      </div>

      {erro && <MensagemErro texto={erro} aoTentarNovamente={() => void recarregar()} />}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Indicador
          Icone={Scroll}
          rotulo="Processos monitorados"
          valor={ativos.length}
          detalhe={`${processos.length - ativos.length} arquivado(s)`}
          para="/processos"
          carregando={carregando}
        />
        <Indicador
          Icone={AlertTriangle}
          rotulo="Alertas urgentes"
          valor={urgentes.length}
          detalhe="não lidos, com prazo"
          para="/notificacoes"
          destaque={urgentes.length > 0}
          carregando={carregando}
        />
        <Indicador
          Icone={FileAudio}
          rotulo="Audiências analisadas"
          valor={prontas}
          detalhe={processando > 0 ? `${processando} em processamento` : `${audiencias.length} no total`}
          para="/audiencias"
          carregando={carregando}
        />
        <Indicador
          Icone={Bell}
          rotulo="Notificações não lidas"
          valor={naoLidas.length}
          detalhe="análises e movimentações"
          para="/notificacoes"
          carregando={carregando}
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-5">
        <Card className="lg:col-span-3">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Precisa da sua atenção</CardTitle>
            <Button variant="ghost" size="sm" className="gap-1" render={<Link to="/notificacoes" />}>
              Ver todas <ArrowRight className="size-3.5" />
            </Button>
          </CardHeader>
          <CardContent>
            {carregando ? (
              <Linhas />
            ) : naoLidas.length === 0 ? (
              <Vazio Icone={CheckCircle2} texto="Nada pendente. Tudo lido." />
            ) : (
              <ul className="divide-y">
                {naoLidas.slice(0, 6).map((n) => (
                  <li key={n.id}>
                    <Link to="/notificacoes" className="flex items-start gap-3 py-3 hover:opacity-80">
                      <span
                        className={cn(
                          "mt-1.5 size-2 shrink-0 rounded-full",
                          n.urgencia === "alta" ? "bg-red-500" : n.urgencia === "media" ? "bg-amber-500" : "bg-primary",
                        )}
                      />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">{n.titulo}</p>
                        <p className="line-clamp-1 text-sm text-muted-foreground">{n.corpo}</p>
                      </div>
                      <span className="shrink-0 text-xs text-muted-foreground">{formatarDataCurta(n.created_at)}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Audiências recentes</CardTitle>
            <Button variant="ghost" size="sm" className="gap-1" render={<Link to="/audiencias" />}>
              Ver todas <ArrowRight className="size-3.5" />
            </Button>
          </CardHeader>
          <CardContent>
            {carregando ? (
              <Linhas />
            ) : audiencias.length === 0 ? (
              <Vazio Icone={FileAudio} texto="Nenhuma audiência enviada." />
            ) : (
              <ul className="divide-y">
                {audiencias.slice(0, 5).map((a) => {
                  const st = STATUS_AUDIENCIA[a.status] ?? STATUS_AUDIENCIA.uploaded
                  return (
                    <li key={a.id}>
                      <Link to={`/audiencias/${a.id}`} className="flex items-center gap-3 py-3 hover:opacity-80">
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm font-medium">{a.titulo || a.nome_arquivo}</p>
                          <p className="text-xs text-muted-foreground">
                            {rotuloArea(a.area_do_direito)} · {formatarDataCurta(a.created_at)}
                          </p>
                        </div>
                        <Badge variant={st.variante} className="shrink-0 gap-1">
                          {st.emAndamento && <Loader2 className="size-3 animate-spin" />}
                          {st.rotulo}
                        </Badge>
                      </Link>
                    </li>
                  )
                })}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Movimentações recentes</CardTitle>
          <Button variant="ghost" size="sm" className="gap-1" render={<Link to="/processos" />}>
            Carteira <ArrowRight className="size-3.5" />
          </Button>
        </CardHeader>
        <CardContent>
          {carregando ? (
            <Linhas />
          ) : recentes.length === 0 ? (
            <Vazio
              Icone={Scroll}
              texto={ativos.length ? "Aguardando a primeira consulta ao tribunal." : "Nenhum processo na carteira."}
            />
          ) : (
            <ul className="divide-y">
              {recentes.map((p) => (
                <li key={p.id}>
                  <Link to={`/processos/${p.id}`} className="flex items-center gap-4 py-3 hover:opacity-80">
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{p.titulo || formatarCNJ(p.numero_cnj)}</p>
                      <p className="truncate text-sm text-muted-foreground">{p.ultima_descricao}</p>
                    </div>
                    <span className="hidden shrink-0 text-xs text-muted-foreground sm:inline">
                      {formatarData(p.ultima_movimentacao!)}
                    </span>
                    {p.ultima_urgencia && <SeloUrgencia urgencia={p.ultima_urgencia} className="shrink-0" />}
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function Indicador({
  Icone, rotulo, valor, detalhe, para, destaque, carregando,
}: {
  Icone: LucideIcon
  rotulo: string
  valor: number
  detalhe: string
  para: string
  destaque?: boolean
  carregando: boolean
}) {
  return (
    <Link to={para}>
      <Card className={cn("py-0 transition-colors hover:border-primary/40", destaque && "border-red-500/40")}>
        <CardContent className="space-y-3 p-5">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">{rotulo}</span>
            <div
              className={cn(
                "flex size-8 items-center justify-center rounded-lg",
                destaque ? "bg-red-500/10 text-red-600" : "bg-secondary text-secondary-foreground",
              )}
            >
              <Icone className="size-4" />
            </div>
          </div>
          {carregando ? (
            <Skeleton className="h-8 w-12" />
          ) : (
            <p className={cn("text-3xl font-semibold tabular-nums", destaque && "text-red-600")}>{valor}</p>
          )}
          <p className="text-xs text-muted-foreground">{detalhe}</p>
        </CardContent>
      </Card>
    </Link>
  )
}

function Linhas() {
  return (
    <div className="space-y-3">
      {[0, 1, 2].map((i) => <Skeleton key={i} className="h-10 w-full" />)}
    </div>
  )
}

function Vazio({ Icone, texto }: { Icone: LucideIcon; texto: string }) {
  return (
    <div className="flex flex-col items-center gap-2 py-8 text-center text-sm text-muted-foreground">
      <Icone className="size-5" />
      {texto}
    </div>
  )
}
