import { NavLink, Outlet } from "react-router-dom"
import { Bell, FileAudio, LogOut, Scale, Scroll } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { useDados, usePolling } from "@/lib/hooks"
import { cn } from "@/lib/utils"

const NAV = [
  { para: "/audiencias", rotulo: "Audiências", Icone: FileAudio },
  { para: "/processos", rotulo: "Processos", Icone: Scroll },
  { para: "/notificacoes", rotulo: "Notificações", Icone: Bell },
]

export function Layout() {
  const { tenant, sair } = useAuth()
  const { dado: resumo, recarregar } = useDados(() => api.resumoNotificacoes(), [])

  // O contador vem de eventos que chegam sozinhos (um worker terminou de
  // classificar uma movimentação), então precisa se atualizar sem ação do
  // usuário — senão o sino mente até alguém navegar.
  usePolling(() => void recarregar(true), true, 15000)

  const naoLidas = resumo?.nao_lidas ?? 0

  return (
    <div className="flex min-h-svh flex-col bg-muted/30">
      <header className="sticky top-0 z-20 border-b bg-background/95 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-6xl items-center gap-3 px-4">
          <div className="flex items-center gap-2 font-semibold">
            <div className="flex size-7 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Scale className="size-4" />
            </div>
            <span className="hidden sm:inline">Meirinho</span>
          </div>

          <nav className="flex flex-1 items-center gap-1 overflow-x-auto">
            {NAV.map(({ para, rotulo, Icone }) => (
              <NavLink
                key={para}
                to={para}
                className={({ isActive }) =>
                  cn(
                    "flex shrink-0 items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-secondary text-secondary-foreground"
                      : "text-muted-foreground hover:bg-muted hover:text-foreground",
                  )
                }
              >
                <Icone className="size-4" />
                {rotulo}
                {para === "/notificacoes" && naoLidas > 0 && (
                  <Badge variant="destructive" className="ml-0.5 h-5 min-w-5 justify-center px-1 text-[11px]">
                    {naoLidas > 99 ? "99+" : naoLidas}
                  </Badge>
                )}
              </NavLink>
            ))}
          </nav>

          <div className="flex items-center gap-2">
            <span className="hidden text-sm text-muted-foreground md:inline">{tenant?.nome}</span>
            <Button variant="ghost" size="icon" onClick={sair} title="Sair" aria-label="Sair">
              <LogOut className="size-4" />
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
