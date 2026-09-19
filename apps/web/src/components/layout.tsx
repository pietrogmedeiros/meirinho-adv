import { useEffect, useRef, useState } from "react"
import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom"
import {
  Bell, BellRing, CheckCheck, ChevronLeft, ChevronRight, FileAudio, KeyRound, LayoutDashboard, LogOut, Menu,
  Scroll, Settings, UserRound, X,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { LogoMeirinho } from "@/components/logo"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { formatarData } from "@/lib/format"
import { useDados, usePolling } from "@/lib/hooks"
import { alertasAtivos, mostrarAlerta } from "@/lib/alertas-navegador"
import { rotaDaNotificacao } from "@/lib/notificacoes"
import type { Notificacao } from "@/lib/tipos"
import { cn } from "@/lib/utils"

interface ItemMenu {
  para: string
  rotulo: string
  Icone: LucideIcon
  fim?: boolean
}

const SECOES: { titulo: string; itens: ItemMenu[] }[] = [
  {
    titulo: "Principais",
    itens: [
      { para: "/", rotulo: "Início", Icone: LayoutDashboard, fim: true },
      { para: "/audiencias", rotulo: "Audiências", Icone: FileAudio },
      { para: "/processos", rotulo: "Processos", Icone: Scroll },
    ],
  },
]

// Notificações não estão no menu (ficam no sino do topo), mas a página existe
// e precisa de título no cabeçalho.
const TITULOS: ItemMenu[] = [
  ...SECOES.flatMap((s) => s.itens),
  { para: "/notificacoes", rotulo: "Notificações", Icone: Bell },
  { para: "/conta", rotulo: "Configurações", Icone: Settings },
]

const CHAVE_RECOLHIDO = "meirinho.menu-recolhido"

function lerRecolhido(): boolean {
  try {
    return localStorage.getItem(CHAVE_RECOLHIDO) === "1"
  } catch {
    return false
  }
}

function iniciais(nome: string): string {
  const partes = nome
    .replace(/^(dra?\.?)\s+/i, "")
    .split(/\s+/)
    .filter(Boolean)
  return ((partes[0]?.[0] ?? "") + (partes.length > 1 ? partes[partes.length - 1][0] : "")).toUpperCase()
}

export function Layout() {
  const { pathname } = useLocation()
  const [recolhido, setRecolhido] = useState(lerRecolhido)
  const [gavetaAberta, setGavetaAberta] = useState(false)

  useEffect(() => {
    try {
      localStorage.setItem(CHAVE_RECOLHIDO, recolhido ? "1" : "0")
    } catch {
      /* preferência é conveniência; sem storage, só não persiste */
    }
  }, [recolhido])

  const titulo = TITULOS.find((n) => (n.fim ? pathname === n.para : pathname.startsWith(n.para)))?.rotulo

  return (
    <div className="flex min-h-svh bg-muted/40">
      {/* Desktop: menu fixo, expandido ou recolhido para só ícones. */}
      <aside
        className={cn(
          "sticky top-0 hidden h-svh shrink-0 flex-col border-r bg-sidebar text-sidebar-foreground transition-[width] duration-200 md:flex",
          recolhido ? "w-[68px]" : "w-60",
        )}
      >
        <ConteudoMenu recolhido={recolhido} aoAlternar={() => setRecolhido((r) => !r)} />
      </aside>

      {/* Celular: gaveta sobre o conteúdo. */}
      {gavetaAberta && (
        <div className="fixed inset-0 z-40 md:hidden">
          <div className="absolute inset-0 bg-black/30" onClick={() => setGavetaAberta(false)} />
          <aside className="absolute inset-y-0 left-0 flex w-64 flex-col bg-sidebar text-sidebar-foreground shadow-xl animate-in slide-in-from-left duration-200">
            <Button
              variant="ghost"
              size="icon"
              className="absolute top-3.5 right-2"
              onClick={() => setGavetaAberta(false)}
              aria-label="Fechar menu"
            >
              <X className="size-4" />
            </Button>
            <ConteudoMenu recolhido={false} aoNavegar={() => setGavetaAberta(false)} />
          </aside>
        </div>
      )}

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-20 flex h-16 items-center gap-2 border-b bg-background/95 px-4 backdrop-blur md:px-8">
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={() => setGavetaAberta(true)}
            aria-label="Abrir menu"
          >
            <Menu className="size-4" />
          </Button>
          <span className="flex-1 font-semibold">{titulo}</span>
          <SinoNotificacoes />
        </header>

        <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6 md:px-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

function ConteudoMenu({
  recolhido, aoAlternar, aoNavegar,
}: {
  recolhido: boolean
  aoAlternar?: () => void
  aoNavegar?: () => void
}) {
  const { tenant, sair } = useAuth()
  const navegar = useNavigate()
  const irPara = (url: string) => {
    aoNavegar?.()
    navegar(url)
  }

  return (
    <>
      <Link to="/" onClick={aoNavegar} className={cn("flex h-16 shrink-0 items-center", recolhido ? "justify-center" : "px-5")}>
        <LogoMeirinho recolhido={recolhido} />
      </Link>

      <nav className="flex-1 overflow-y-auto py-2">
        {SECOES.map((secao) => (
          <div key={secao.titulo} className="mb-4">
            {recolhido ? (
              <div className="mx-4 mb-2 h-px bg-sidebar-border" />
            ) : (
              <p className="mb-1.5 px-5 text-[11px] font-medium tracking-[0.08em] text-muted-foreground uppercase">
                {secao.titulo}
              </p>
            )}
            {secao.itens.map(({ para, rotulo, Icone, fim }) => (
              <NavLink
                key={para}
                to={para}
                end={fim}
                onClick={aoNavegar}
                title={recolhido ? rotulo : undefined}
                className={({ isActive }) =>
                  cn(
                    "relative flex h-11 items-center gap-3 text-[15px] transition-colors",
                    recolhido ? "justify-center" : "px-5",
                    isActive
                      ? "bg-sidebar-accent font-semibold text-sidebar-accent-foreground"
                      : "hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground",
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    {isActive && <span className="absolute inset-y-0 left-0 w-[3px] bg-sidebar-primary" />}
                    <Icone className="size-[18px] shrink-0" strokeWidth={isActive ? 2.2 : 1.8} />
                    {!recolhido && <span className="truncate">{rotulo}</span>}
                  </>
                )}
              </NavLink>
            ))}
          </div>
        ))}
      </nav>

      <div className="border-t border-sidebar-border">
        <div className={cn("flex items-center gap-1 py-3", recolhido ? "flex-col px-2" : "px-3")}>
          <DropdownMenu>
            <DropdownMenuTrigger
              className={cn(
                "flex min-w-0 flex-1 items-center gap-3 rounded-lg p-2 text-left transition-colors outline-none hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-sidebar-ring data-popup-open:bg-sidebar-accent",
                recolhido && "justify-center",
              )}
              title={recolhido ? tenant?.nome : undefined}
            >
              <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-sidebar-primary text-xs font-semibold text-sidebar-primary-foreground">
                {iniciais(tenant?.nome ?? "")}
              </span>
              {!recolhido && (
                <span className="min-w-0 leading-tight">
                  <span className="block truncate text-sm font-semibold text-sidebar-accent-foreground">{tenant?.nome}</span>
                  <span className="block truncate text-xs text-muted-foreground">OAB {tenant?.oab}</span>
                </span>
              )}
            </DropdownMenuTrigger>
            <DropdownMenuContent side={recolhido ? "right" : "top"} align="start" className="w-60">
              <div className="px-2 py-1.5">
                <p className="truncate text-sm font-medium">{tenant?.nome}</p>
                <p className="truncate text-xs text-muted-foreground">{tenant?.email}</p>
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => irPara("/conta?aba=perfil")}>
                <UserRound className="size-4" /> Editar dados
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => irPara("/conta?aba=seguranca")}>
                <KeyRound className="size-4" /> Redefinir senha
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => irPara("/conta?aba=notificacoes")}>
                <BellRing className="size-4" /> Ativar notificações
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={sair}>
                <LogOut className="size-4" /> Sair da conta
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          <Button
            variant="ghost"
            size="icon"
            onClick={() => irPara("/conta")}
            title="Configurações"
            aria-label="Configurações"
            className="shrink-0 text-muted-foreground hover:text-sidebar-accent-foreground"
          >
            <Settings className="size-4" />
          </Button>
        </div>

        {recolhido ? (
          <button
            type="button"
            onClick={sair}
            title="Sair da conta"
            aria-label="Sair da conta"
            className="flex h-10 w-full items-center justify-center hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <LogOut className="size-4" />
          </button>
        ) : (
          <button
            type="button"
            onClick={sair}
            className="h-10 w-full text-xs font-semibold tracking-[0.12em] text-sidebar-accent-foreground uppercase hover:bg-sidebar-accent"
          >
            Sair da conta
          </button>
        )}

        {aoAlternar && (
          <button
            type="button"
            onClick={aoAlternar}
            aria-label={recolhido ? "Expandir menu" : "Recolher menu"}
            title={recolhido ? "Expandir menu" : undefined}
            className="flex h-11 w-full items-center justify-center gap-1.5 border-t border-sidebar-border text-sm text-muted-foreground hover:text-sidebar-accent-foreground"
          >
            {recolhido ? <ChevronRight className="size-4" /> : <><ChevronLeft className="size-4" /> Recolher</>}
          </button>
        )}
      </div>
    </>
  )
}

/**
 * Sino do topo com as notificações num painel que abre ao passar o mouse.
 * Também abre no clique, para quem usa toque ou teclado.
 */
function SinoNotificacoes() {
  const navegar = useNavigate()
  const { pathname } = useLocation()
  const [aberto, setAberto] = useState(false)
  const fechar = useRef<ReturnType<typeof setTimeout> | null>(null)

  const { dado: resumo, recarregar: recarregarResumo } = useDados(() => api.resumoNotificacoes(), [])
  const { dado: itens, carregando, recarregar: recarregarItens } = useDados(() => api.notificacoes(), [])

  // O contador vem de eventos que chegam sozinhos (um worker terminou de
  // classificar uma movimentação), então precisa se atualizar sem ação do
  // usuário — senão o sino mente até alguém navegar.
  usePolling(() => void recarregarResumo(true), true, 15000)
  useEffect(() => void recarregarResumo(true), [pathname, recarregarResumo])

  const naoLidas = resumo?.nao_lidas ?? 0

  // Alerta do navegador quando o contador sobe. Compara com o valor anterior,
  // não com zero: a primeira leitura depois do login não deve disparar nada.
  const anterior = useRef<number | null>(null)
  useEffect(() => {
    if (resumo == null) return
    const antes = anterior.current
    anterior.current = resumo.nao_lidas
    if (antes == null || resumo.nao_lidas <= antes || !alertasAtivos()) return
    void api.notificacoes(true).then((novas) => {
      const n = novas[0]
      if (!n) return
      const extra = resumo.nao_lidas - antes > 1 ? ` (+${resumo.nao_lidas - antes - 1})` : ""
      mostrarAlerta(n.titulo + extra, n.corpo, () => void abrirNotificacao(n))
    })
    // abrirNotificacao muda a cada render; o gatilho é só o contador.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resumo])

  function abrir() {
    if (fechar.current) clearTimeout(fechar.current)
    if (!aberto) void recarregarItens(true)
    setAberto(true)
  }
  // Pequena espera ao sair: o mouse atravessa o vão entre o sino e o painel.
  function agendarFechar() {
    if (fechar.current) clearTimeout(fechar.current)
    fechar.current = setTimeout(() => setAberto(false), 180)
  }

  async function abrirNotificacao(n: Notificacao) {
    setAberto(false)
    if (!n.lida) void api.marcarLida(n.id).then(() => recarregarResumo(true)).catch(() => {})
    const destino = await rotaDaNotificacao(n)
    if (destino) navegar(destino)
  }

  async function marcarTodas() {
    await api.marcarTodasLidas().catch(() => {})
    void recarregarResumo(true)
    void recarregarItens(true)
  }

  const lista = (itens ?? []).slice(0, 8)

  return (
    <div className="relative" onMouseEnter={abrir} onMouseLeave={agendarFechar}>
      <Button
        variant="ghost"
        size="icon"
        className="relative"
        aria-label={naoLidas ? `Notificações (${naoLidas} não lidas)` : "Notificações"}
        aria-expanded={aberto}
        onClick={() => (aberto ? setAberto(false) : abrir())}
      >
        <Bell className="size-[18px]" />
        {naoLidas > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-semibold text-white ring-2 ring-background">
            {naoLidas > 99 ? "99+" : naoLidas}
          </span>
        )}
      </Button>

      {aberto && (
        <div className="absolute top-full right-0 z-50 pt-2">
          <div className="w-[min(380px,calc(100vw-2rem))] overflow-hidden rounded-xl border bg-popover text-popover-foreground shadow-lg animate-in fade-in-0 zoom-in-95 duration-100">
            <div className="flex items-center justify-between border-b px-4 py-3">
              <p className="text-sm font-semibold">
                Notificações
                {naoLidas > 0 && <span className="ml-1.5 font-normal text-muted-foreground">({naoLidas} não lidas)</span>}
              </p>
              {naoLidas > 0 && (
                <button
                  type="button"
                  onClick={marcarTodas}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                >
                  <CheckCheck className="size-3.5" />
                  Marcar como lidas
                </button>
              )}
            </div>

            <div className="max-h-96 overflow-y-auto">
              {carregando && !itens ? (
                <p className="px-4 py-8 text-center text-sm text-muted-foreground">Carregando…</p>
              ) : lista.length === 0 ? (
                <p className="px-4 py-8 text-center text-sm text-muted-foreground">Nenhuma notificação.</p>
              ) : (
                lista.map((n) => (
                  <button
                    key={n.id}
                    type="button"
                    onClick={() => void abrirNotificacao(n)}
                    className="flex w-full gap-3 border-b px-4 py-3 text-left last:border-b-0 hover:bg-muted/60"
                  >
                    <span
                      className={cn(
                        "mt-1.5 size-2 shrink-0 rounded-full",
                        n.lida
                          ? "bg-transparent"
                          : n.urgencia === "alta"
                            ? "bg-red-500"
                            : n.urgencia === "media"
                              ? "bg-amber-500"
                              : "bg-foreground",
                      )}
                    />
                    <div className="min-w-0 flex-1">
                      <p className={cn("line-clamp-1 text-sm", n.lida ? "text-muted-foreground" : "font-medium")}>
                        {n.titulo}
                      </p>
                      <p className="line-clamp-2 text-xs text-muted-foreground">{n.corpo}</p>
                      <p className="mt-1 text-[11px] text-muted-foreground">{formatarData(n.created_at)}</p>
                    </div>
                  </button>
                ))
              )}
            </div>

            <Link
              to="/notificacoes"
              onClick={() => setAberto(false)}
              className="block border-t py-2.5 text-center text-sm font-medium hover:bg-muted/60"
            >
              Ver todas
            </Link>
          </div>
        </div>
      )}
    </div>
  )
}
