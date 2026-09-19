import { useState } from "react"
import { MarcaMeirinho } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { useAuth } from "@/lib/auth"

export function LoginPage() {
  const { entrar, cadastrar } = useAuth()
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)
  const [modo, setModo] = useState<"entrar" | "criar">("entrar")

  async function comTratamento(fn: () => Promise<void>) {
    setErro(null)
    setEnviando(true)
    try {
      await fn()
    } catch (e) {
      setErro(e instanceof Error ? e.message : "Não foi possível continuar.")
    } finally {
      setEnviando(false)
    }
  }

  function aoEntrar(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    void comTratamento(() =>
      entrar(String(f.get("email") ?? ""), String(f.get("senha") ?? "")),
    )
  }

  function aoCadastrar(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    void comTratamento(() =>
      cadastrar({
        nome: String(f.get("nome") ?? ""),
        oab: String(f.get("oab") ?? ""),
        email: String(f.get("email") ?? ""),
        senha: String(f.get("senha") ?? ""),
      }),
    )
  }

  return (
    <div className="grid min-h-svh bg-muted/40 lg:grid-cols-2">
      <PainelInstitucional />

      <div className="flex flex-col items-center justify-center gap-6 p-6">
        {/* No celular o painel escuro some; a marca vem para cima do formulário. */}
        <div className="flex items-center gap-2.5 lg:hidden">
          <MarcaMeirinho className="size-8" />
          <span className="font-logo text-3xl leading-none">Meirinho</span>
        </div>

        <Card className="w-full max-w-sm shadow-sm">
          <CardHeader className="space-y-1">
            <h2 className="text-2xl font-semibold tracking-tight">{modo === "entrar" ? "Entrar" : "Criar conta"}</h2>
            <p className="text-sm text-muted-foreground">
              {modo === "entrar"
                ? "Use o e-mail e a senha do seu escritório."
                : "Cadastre-se com o seu número de inscrição na OAB."}
            </p>
          </CardHeader>

          <CardContent>
            {erro && (
              <Alert variant="destructive" className="mb-4">
                <AlertDescription>{erro}</AlertDescription>
              </Alert>
            )}

            {modo === "entrar" ? (
              <form onSubmit={aoEntrar} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="email-entrar">E-mail</Label>
                  <Input
                    id="email-entrar"
                    name="email"
                    type="email"
                    required
                    autoComplete="email"
                    placeholder="voce@escritorio.adv.br"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="senha-entrar">Senha</Label>
                  <Input
                    id="senha-entrar"
                    name="senha"
                    type="password"
                    required
                    autoComplete="current-password"
                  />
                </div>
                <Button type="submit" className="h-10 w-full" disabled={enviando}>
                  {enviando ? "Entrando…" : "Entrar"}
                </Button>
              </form>
            ) : (
              <form onSubmit={aoCadastrar} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="nome">Nome</Label>
                  <Input id="nome" name="nome" required autoComplete="name" placeholder="Dra. Ana Costa" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="oab">OAB</Label>
                  <Input id="oab" name="oab" required placeholder="SP 123.456" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="email-criar">E-mail</Label>
                  <Input id="email-criar" name="email" type="email" required autoComplete="email" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="senha-criar">Senha</Label>
                  <Input
                    id="senha-criar"
                    name="senha"
                    type="password"
                    required
                    minLength={8}
                    autoComplete="new-password"
                    placeholder="Ao menos 8 caracteres"
                  />
                </div>
                <Button type="submit" className="h-10 w-full" disabled={enviando}>
                  {enviando ? "Criando…" : "Criar conta"}
                </Button>
              </form>
            )}

            <p className="mt-5 text-center text-sm text-muted-foreground">
              {modo === "entrar" ? "Ainda não tem conta? " : "Já tem conta? "}
              <button
                type="button"
                className="font-medium text-foreground underline-offset-4 hover:underline"
                onClick={() => {
                  setErro(null)
                  setModo(modo === "entrar" ? "criar" : "entrar")
                }}
              >
                {modo === "entrar" ? "Criar conta" : "Entrar"}
              </button>
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

const PONTOS = [
  "Audiência gravada vira resumo, estratégia e pontos críticos.",
  "Movimentações do tribunal classificadas por urgência.",
  "Alerta quando uma intimação abre prazo.",
]

function PainelInstitucional() {
  return (
    <aside className="relative hidden flex-col justify-between overflow-hidden bg-neutral-950 p-12 text-neutral-100 lg:flex">
      {/* Marca em escala de fachada, quase apagada: textura, não ilustração. */}
      <MarcaMeirinho className="pointer-events-none absolute -right-24 -bottom-16 size-[560px] text-white/[0.035]" />

      <div className="flex items-center gap-2.5">
        <MarcaMeirinho className="size-8 text-white" />
        <span className="font-logo text-3xl leading-none text-white">Meirinho</span>
      </div>

      <div className="relative max-w-lg space-y-8">
        <p className="text-xs font-medium tracking-[0.2em] text-neutral-400 uppercase">
          Para a advocacia autônoma
        </p>
        <h1 className="font-logo text-5xl leading-[1.05] text-white">
          Da sala de audiência ao prazo cumprido.
        </h1>
        <ul className="space-y-3 text-[15px] text-neutral-300">
          {PONTOS.map((p) => (
            <li key={p} className="flex gap-3">
              <span className="mt-2.5 h-px w-4 shrink-0 bg-neutral-500" />
              {p}
            </li>
          ))}
        </ul>
      </div>

      <p className="relative text-xs text-neutral-500">
        © {new Date().getFullYear()} Meirinho · Seus dados ficam isolados por escritório.
      </p>
    </aside>
  )
}
