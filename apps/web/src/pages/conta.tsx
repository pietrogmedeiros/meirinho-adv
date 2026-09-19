import { useEffect, useState } from "react"
import { useSearchParams } from "react-router-dom"
import { Bell, KeyRound, Loader2, UserRound } from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { MensagemErro } from "@/components/estados"
import { ativarAlertas, alertasAtivos, desativarAlertas, suportaAlertas } from "@/lib/alertas-navegador"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { useDados } from "@/lib/hooks"
import type { PreferenciasNotificacao, Urgencia } from "@/lib/tipos"
import { cn } from "@/lib/utils"

type Aba = "perfil" | "seguranca" | "notificacoes"

const ABAS: { id: Aba; rotulo: string; Icone: LucideIcon }[] = [
  { id: "perfil", rotulo: "Perfil", Icone: UserRound },
  { id: "seguranca", rotulo: "Senha e segurança", Icone: KeyRound },
  { id: "notificacoes", rotulo: "Notificações", Icone: Bell },
]

export function ContaPage() {
  // A aba vive na URL: o menu do usuário leva direto a "Alterar senha" etc.
  const [params, setParams] = useSearchParams()
  const aba: Aba = ABAS.some((a) => a.id === params.get("aba")) ? (params.get("aba") as Aba) : "perfil"

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Configurações</h1>
        <p className="text-sm text-muted-foreground">Seus dados, sua senha e como você quer ser avisado.</p>
      </div>

      <div className="grid gap-6 md:grid-cols-[200px_1fr]">
        <nav className="flex gap-1 overflow-x-auto md:flex-col">
          {ABAS.map(({ id, rotulo, Icone }) => (
            <button
              key={id}
              type="button"
              onClick={() => setParams({ aba: id }, { replace: true })}
              className={cn(
                "flex shrink-0 items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm transition-colors",
                aba === id ? "bg-background font-medium shadow-sm ring-1 ring-border" : "text-muted-foreground hover:text-foreground",
              )}
            >
              <Icone className="size-4" />
              {rotulo}
            </button>
          ))}
        </nav>

        <div className="max-w-2xl">
          {aba === "perfil" && <Perfil />}
          {aba === "seguranca" && <Seguranca />}
          {aba === "notificacoes" && <Notificacoes />}
        </div>
      </div>
    </div>
  )
}

function Perfil() {
  const { tenant, atualizarTenant } = useAuth()
  const [salvando, setSalvando] = useState(false)

  async function salvar(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    setSalvando(true)
    try {
      const t = await api.atualizarPerfil({
        nome: String(f.get("nome") ?? ""),
        oab: String(f.get("oab") ?? ""),
        email: String(f.get("email") ?? ""),
      })
      atualizarTenant(t)
      toast.success("Perfil atualizado")
    } catch (err) {
      toast.error("Não foi possível salvar", { description: err instanceof Error ? err.message : undefined })
    } finally {
      setSalvando(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Perfil</CardTitle>
        <CardDescription>Como você aparece no Meirinho. O e-mail também é o seu login.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={salvar} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="nome">Nome</Label>
            <Input id="nome" name="nome" defaultValue={tenant?.nome} required autoComplete="name" />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="oab">Inscrição na OAB</Label>
              <Input id="oab" name="oab" defaultValue={tenant?.oab} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">E-mail</Label>
              <Input id="email" name="email" type="email" defaultValue={tenant?.email} required autoComplete="email" />
            </div>
          </div>
          <div className="flex justify-end">
            <Button type="submit" disabled={salvando} className="gap-2">
              {salvando && <Loader2 className="size-4 animate-spin" />}
              Salvar alterações
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

function Seguranca() {
  const [salvando, setSalvando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  async function salvar(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = e.currentTarget
    const f = new FormData(form)
    const nova = String(f.get("nova") ?? "")
    if (nova !== String(f.get("confirmacao") ?? "")) {
      setErro("A confirmação não confere com a nova senha.")
      return
    }
    setErro(null)
    setSalvando(true)
    try {
      await api.trocarSenha({ senha_atual: String(f.get("atual") ?? ""), nova_senha: nova })
      toast.success("Senha alterada", { description: "Use a nova senha no próximo acesso." })
      form.reset()
    } catch (err) {
      setErro(err instanceof Error ? err.message : "Não foi possível alterar a senha.")
    } finally {
      setSalvando(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Redefinir senha</CardTitle>
        <CardDescription>Por segurança, confirme a senha atual antes de definir a nova.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={salvar} className="space-y-4">
          {erro && <MensagemErro texto={erro} />}
          <div className="space-y-2">
            <Label htmlFor="atual">Senha atual</Label>
            <Input id="atual" name="atual" type="password" required autoComplete="current-password" />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="nova">Nova senha</Label>
              <Input id="nova" name="nova" type="password" required minLength={8} autoComplete="new-password" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirmacao">Confirme a nova senha</Label>
              <Input id="confirmacao" name="confirmacao" type="password" required minLength={8} autoComplete="new-password" />
            </div>
          </div>
          <p className="text-xs text-muted-foreground">Ao menos 8 caracteres.</p>
          <div className="flex justify-end">
            <Button type="submit" disabled={salvando} className="gap-2">
              {salvando && <Loader2 className="size-4 animate-spin" />}
              Alterar senha
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

const OPCOES_URGENCIA: { valor: Urgencia; rotulo: string }[] = [
  { valor: "nenhuma", rotulo: "Todas as movimentações" },
  { valor: "baixa", rotulo: "Urgência baixa ou maior" },
  { valor: "media", rotulo: "Urgência média ou alta" },
  { valor: "alta", rotulo: "Só urgência alta (prazos)" },
]

function Notificacoes() {
  const { dado, erro, carregando, recarregar } = useDados(() => api.preferencias(), [])
  const [prefs, setPrefs] = useState<PreferenciasNotificacao | null>(null)
  const [salvando, setSalvando] = useState(false)
  const [navegador, setNavegador] = useState(alertasAtivos)

  useEffect(() => {
    if (dado) setPrefs(dado)
  }, [dado])

  async function alternarNavegador(ligar: boolean) {
    if (!ligar) {
      desativarAlertas()
      setNavegador(false)
      return
    }
    const ok = await ativarAlertas()
    setNavegador(ok)
    if (!ok) {
      toast.error("O navegador bloqueou as notificações", {
        description: "Libere nas configurações do site (ícone ao lado do endereço) e tente de novo.",
      })
    }
  }

  async function salvar() {
    if (!prefs) return
    setSalvando(true)
    try {
      setPrefs(await api.salvarPreferencias(prefs))
      toast.success("Preferências salvas")
    } catch (err) {
      toast.error("Não foi possível salvar", { description: err instanceof Error ? err.message : undefined })
    } finally {
      setSalvando(false)
    }
  }

  if (erro) return <MensagemErro texto={erro} aoTentarNovamente={() => void recarregar()} />
  if (carregando || !prefs) return <Skeleton className="h-96 w-full" />

  const disp = prefs.canais_disponiveis ?? { email: false, whatsapp: false }
  const mudar = (p: Partial<PreferenciasNotificacao>) => setPrefs({ ...prefs, ...p })

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>O que gera alerta</CardTitle>
          <CardDescription>
            Tudo continua registrado no histórico; isto decide só o que chega como notificação.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="space-y-2">
            <Label htmlFor="urgencia">Movimentações de processo</Label>
            <Select
              items={OPCOES_URGENCIA.map((o) => ({ value: o.valor, label: o.rotulo }))}
              value={prefs.urgencia_minima}
              onValueChange={(v) => v && mudar({ urgencia_minima: v as Urgencia })}
            >
              <SelectTrigger id="urgencia" className="w-full sm:w-80">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {OPCOES_URGENCIA.map((o) => (
                  <SelectItem key={o.valor} value={o.valor}>{o.rotulo}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Quando a classificação automática fica incerta, você é avisado mesmo abaixo deste nível.
            </p>
          </div>
          <Opcao
            titulo="Análise de audiência concluída"
            descricao="Aviso quando o resumo e a estratégia de uma audiência ficam prontos."
            ligado={prefs.alertar_audiencias}
            aoMudar={(v) => mudar({ alertar_audiencias: v })}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Onde avisar</CardTitle>
          <CardDescription>As notificações sempre aparecem no sino do Meirinho.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <Opcao
            titulo="Notificações no navegador"
            descricao={
              suportaAlertas()
                ? "Alerta do sistema neste computador quando algo novo chegar, com o Meirinho aberto."
                : "Este navegador não permite notificações."
            }
            ligado={navegador}
            desabilitado={!suportaAlertas()}
            aoMudar={(v) => void alternarNavegador(v)}
          />
          <Opcao
            titulo="E-mail"
            descricao={`Enviado para o e-mail da sua conta.${disp.email ? "" : " Ainda não ativo neste servidor: a preferência fica salva."}`}
            ligado={prefs.canal_email}
            aoMudar={(v) => mudar({ canal_email: v })}
          />
          <div className="space-y-3">
            <Opcao
              titulo="WhatsApp"
              descricao={`Mensagem no seu celular.${disp.whatsapp ? "" : " Ainda não ativo neste servidor: a preferência fica salva."}`}
              ligado={prefs.canal_whatsapp}
              aoMudar={(v) => mudar({ canal_whatsapp: v })}
            />
            {prefs.canal_whatsapp && (
              <div className="space-y-2 pl-0 sm:pl-1">
                <Label htmlFor="telefone">Celular com DDD</Label>
                <Input
                  id="telefone"
                  value={prefs.telefone}
                  onChange={(e) => mudar({ telefone: e.target.value })}
                  placeholder="+55 11 91234-5678"
                  inputMode="tel"
                  className="sm:w-64"
                />
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button onClick={salvar} disabled={salvando} className="gap-2">
          {salvando && <Loader2 className="size-4 animate-spin" />}
          Salvar preferências
        </Button>
      </div>
    </div>
  )
}

function Opcao({
  titulo, descricao, ligado, aoMudar, desabilitado,
}: {
  titulo: string
  descricao: string
  ligado: boolean
  aoMudar: (v: boolean) => void
  desabilitado?: boolean
}) {
  return (
    <div className="flex items-start justify-between gap-6">
      <div className="space-y-0.5">
        <p className="text-sm font-medium">{titulo}</p>
        <p className="text-sm text-muted-foreground">{descricao}</p>
      </div>
      <Interruptor ligado={ligado} aoMudar={aoMudar} rotulo={titulo} desabilitado={desabilitado} />
    </div>
  )
}

function Interruptor({
  ligado, aoMudar, rotulo, desabilitado,
}: {
  ligado: boolean
  aoMudar: (v: boolean) => void
  rotulo: string
  desabilitado?: boolean
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={ligado}
      aria-label={rotulo}
      disabled={desabilitado}
      onClick={() => aoMudar(!ligado)}
      className={cn(
        "relative mt-0.5 inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none disabled:opacity-50",
        ligado ? "bg-primary" : "bg-input",
      )}
    >
      <span
        className={cn(
          "inline-block size-5 rounded-full bg-background shadow transition-transform",
          ligado ? "translate-x-[22px]" : "translate-x-0.5",
        )}
      />
    </button>
  )
}
