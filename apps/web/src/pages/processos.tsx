import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Loader2, Plus, Scroll, Search } from "lucide-react"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { EstadoVazio, MensagemErro } from "@/components/estados"
import { SeloUrgencia } from "@/components/urgencia"
import { api } from "@/lib/api"
import { formatarCNJ, formatarDataCurta, rotuloArea } from "@/lib/format"
import { useDados, usePolling } from "@/lib/hooks"
import { AREAS, type AreaDoDireito } from "@/lib/tipos"

type Filtro = "ativos" | "arquivados" | "todos"

export function ProcessosPage() {
  const navegar = useNavigate()
  const { dado: processos, erro, carregando, recarregar } = useDados(() => api.processos(), [])
  const [busca, setBusca] = useState("")
  const [filtro, setFiltro] = useState<Filtro>("ativos")

  // Um processo recém-cadastrado ainda não passou pelo poller: enquanto houver
  // algum assim, a lista se atualiza para mostrar o histórico chegando.
  const aguardando = (processos ?? []).some((p) => p.status === "ativo" && p.total_movimentacoes === 0)
  usePolling(() => void recarregar(true), aguardando, 5000)

  const termo = busca.trim().toLowerCase()
  const digitos = termo.replace(/\D/g, "")
  const visiveis = (processos ?? []).filter((p) => {
    if (filtro === "ativos" && p.status !== "ativo") return false
    if (filtro === "arquivados" && p.status !== "arquivado") return false
    if (!termo) return true
    return (
      p.titulo.toLowerCase().includes(termo) ||
      p.tribunal.toLowerCase().includes(termo) ||
      (digitos.length > 0 && p.numero_cnj.includes(digitos))
    )
  })

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Processos</h1>
          <p className="text-sm text-muted-foreground">
            Os processos ativos são consultados no tribunal periodicamente.
          </p>
        </div>
        <DialogNovoProcesso aoCriar={() => void recarregar(true)} />
      </div>

      {erro && <MensagemErro texto={erro} aoTentarNovamente={() => void recarregar()} />}

      {!carregando && (processos ?? []).length > 0 && (
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative min-w-56 flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={busca}
              onChange={(e) => setBusca(e.target.value)}
              placeholder="Buscar por parte, tribunal ou número CNJ"
              className="bg-background pl-8"
            />
          </div>
          <Tabs value={filtro} onValueChange={(v) => setFiltro(v as Filtro)}>
            <TabsList>
              <TabsTrigger value="ativos">Ativos</TabsTrigger>
              <TabsTrigger value="arquivados">Arquivados</TabsTrigger>
              <TabsTrigger value="todos">Todos</TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      )}

      {carregando ? (
        <div className="space-y-3">
          {[0, 1, 2].map((i) => <Skeleton key={i} className="h-14 w-full" />)}
        </div>
      ) : (processos ?? []).length === 0 && !erro ? (
        <EstadoVazio
          Icone={Scroll}
          titulo="Nenhum processo na carteira"
          descricao="Cadastre o número CNJ de um processo para acompanhar as movimentações e receber alerta quando algo abrir prazo."
        />
      ) : visiveis.length === 0 ? (
        <p className="py-10 text-center text-sm text-muted-foreground">Nenhum processo corresponde ao filtro.</p>
      ) : (
        <Card className="py-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="pl-4">Processo</TableHead>
                <TableHead className="hidden lg:table-cell">Última movimentação</TableHead>
                <TableHead className="hidden md:table-cell">Área</TableHead>
                <TableHead className="pr-4 text-right">Urgência</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {visiveis.map((p) => (
                <TableRow
                  key={p.id}
                  className="cursor-pointer"
                  onClick={() => navegar(`/processos/${p.id}`)}
                >
                  <TableCell className="max-w-0 py-3 pl-4 lg:w-[40%]">
                    <div className="flex items-center gap-2">
                      <p className="truncate font-medium">{p.titulo || formatarCNJ(p.numero_cnj)}</p>
                      {p.status === "arquivado" && <Badge variant="outline">Arquivado</Badge>}
                    </div>
                    <p className="truncate text-xs text-muted-foreground">
                      <span className="font-mono">{formatarCNJ(p.numero_cnj)}</span>
                      {p.tribunal && ` · ${p.tribunal}`}
                    </p>
                  </TableCell>
                  <TableCell className="hidden max-w-0 py-3 lg:table-cell">
                    {p.ultima_movimentacao ? (
                      <>
                        <p className="truncate text-sm">{p.ultima_descricao}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatarDataCurta(p.ultima_movimentacao)} · {p.total_movimentacoes}{" "}
                          {p.total_movimentacoes === 1 ? "movimentação" : "movimentações"}
                        </p>
                      </>
                    ) : (
                      <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
                        {p.status === "ativo" && <Loader2 className="size-3 animate-spin" />}
                        {p.status === "ativo" ? "Consultando o tribunal…" : "—"}
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="hidden md:table-cell">{rotuloArea(p.area_do_direito)}</TableCell>
                  <TableCell className="pr-4 text-right">
                    {p.ultima_urgencia ? <SeloUrgencia urgencia={p.ultima_urgencia} /> : <span className="text-muted-foreground">—</span>}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}
    </div>
  )
}

/** Máscara progressiva do CNJ: NNNNNNN-DD.AAAA.J.TR.OOOO */
function mascararCNJ(v: string): string {
  const d = v.replace(/\D/g, "").slice(0, 20)
  const partes = [d.slice(0, 7), d.slice(7, 9), d.slice(9, 13), d.slice(13, 14), d.slice(14, 16), d.slice(16, 20)]
  const seps = ["", "-", ".", ".", ".", "."]
  return partes.reduce((acc, p, i) => (p ? acc + seps[i] + p : acc), "")
}

function DialogNovoProcesso({ aoCriar }: { aoCriar: () => void }) {
  const [aberto, setAberto] = useState(false)
  const [enviando, setEnviando] = useState(false)
  const [cnj, setCnj] = useState("")
  const [titulo, setTitulo] = useState("")
  const [tribunal, setTribunal] = useState("")
  const [area, setArea] = useState<AreaDoDireito>("civel")

  const cnjCompleto = cnj.replace(/\D/g, "").length === 20

  async function submeter(e: React.FormEvent) {
    e.preventDefault()
    setEnviando(true)
    try {
      await api.criarProcesso({ numero_cnj: cnj, titulo, tribunal, area_do_direito: area })
      toast.success("Processo cadastrado", {
        description: "O histórico de movimentações chega em instantes.",
      })
      setAberto(false)
      setCnj("")
      setTitulo("")
      setTribunal("")
      aoCriar()
    } catch (err) {
      toast.error("Não foi possível cadastrar", {
        description: err instanceof Error ? err.message : undefined,
      })
    } finally {
      setEnviando(false)
    }
  }

  return (
    <Dialog open={aberto} onOpenChange={setAberto}>
      <DialogTrigger render={<Button className="gap-2" />}>
        <Plus className="size-4" />
        Novo processo
      </DialogTrigger>
      <DialogContent>
        <form onSubmit={submeter}>
          <DialogHeader>
            <DialogTitle>Novo processo</DialogTitle>
            <DialogDescription>
              Informe o número no padrão CNJ; a consulta ao tribunal começa em seguida.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="cnj">Número CNJ</Label>
              <Input
                id="cnj"
                inputMode="numeric"
                required
                value={cnj}
                onChange={(e) => setCnj(mascararCNJ(e.target.value))}
                placeholder="0000000-00.0000.0.00.0000"
                className="font-mono"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="titulo-processo">Título</Label>
              <Input
                id="titulo-processo"
                value={titulo}
                onChange={(e) => setTitulo(e.target.value)}
                placeholder="Ex.: Silva x Banco Exemplo"
              />
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="tribunal">Tribunal</Label>
                <Input
                  id="tribunal"
                  value={tribunal}
                  onChange={(e) => setTribunal(e.target.value)}
                  placeholder="Ex.: TJSP"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="area-processo">Área do direito</Label>
                <Select
                  items={AREAS.map((a) => ({ value: a.valor, label: a.rotulo }))}
                  value={area}
                  onValueChange={(v) => v && setArea(v as AreaDoDireito)}
                >
                  <SelectTrigger id="area-processo" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {AREAS.map((a) => (
                      <SelectItem key={a.valor} value={a.valor}>{a.rotulo}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button type="submit" disabled={enviando || !cnjCompleto} className="gap-2">
              {enviando && <Loader2 className="size-4 animate-spin" />}
              {enviando ? "Cadastrando…" : "Cadastrar"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
