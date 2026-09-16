import { useState } from "react"
import { Scale } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { useAuth } from "@/lib/auth"

export function LoginPage() {
  const { entrar, cadastrar } = useAuth()
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)

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
    <div className="flex min-h-svh items-center justify-center bg-muted/40 p-4">
      <div className="w-full max-w-md space-y-6">
        <div className="flex flex-col items-center gap-2 text-center">
          <div className="flex size-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <Scale className="size-6" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Meirinho</h1>
          <p className="text-sm text-muted-foreground">
            Audiências transcritas e processos monitorados, num lugar só.
          </p>
        </div>

        <Card>
          <Tabs defaultValue="entrar">
            <CardHeader>
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="entrar">Entrar</TabsTrigger>
                <TabsTrigger value="criar">Criar conta</TabsTrigger>
              </TabsList>
            </CardHeader>

            <CardContent>
              {erro && (
                <Alert variant="destructive" className="mb-4">
                  <AlertDescription>{erro}</AlertDescription>
                </Alert>
              )}

              <TabsContent value="entrar" className="m-0">
                <form onSubmit={aoEntrar} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="email-entrar">E-mail</Label>
                    <Input id="email-entrar" name="email" type="email" required autoComplete="email" />
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
                  <Button type="submit" className="w-full" disabled={enviando}>
                    {enviando ? "Entrando…" : "Entrar"}
                  </Button>
                </form>
              </TabsContent>

              <TabsContent value="criar" className="m-0">
                <form onSubmit={aoCadastrar} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="nome">Nome</Label>
                    <Input id="nome" name="nome" required autoComplete="name" />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="oab">OAB</Label>
                    <Input id="oab" name="oab" required placeholder="SP123456" />
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
                    />
                  </div>
                  <Button type="submit" className="w-full" disabled={enviando}>
                    {enviando ? "Criando…" : "Criar conta"}
                  </Button>
                </form>
              </TabsContent>
            </CardContent>
          </Tabs>
        </Card>

        <p className="text-center text-xs text-muted-foreground">
          Cada advogado é um tenant isolado: seus dados não são visíveis a mais ninguém.
        </p>
      </div>
    </div>
  )
}
