import { useState } from "react"
import { Link } from "react-router-dom"
import { FileAudio, Loader2, Upload } from "lucide-react"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { api } from "@/lib/api"
import { formatarData, STATUS_AUDIENCIA } from "@/lib/format"
import { useDados, usePolling } from "@/lib/hooks"
import { AREAS, type AreaDoDireito } from "@/lib/tipos"
import { EstadoVazio, MensagemErro } from "@/components/estados"

export function AudienciasPage() {
  const { dado: audiencias, erro, carregando, recarregar } = useDados(() => api.audiencias(), [])

  // Enquanto alguma audiência estiver no meio do pipeline, a lista se atualiza
  // sozinha — o trabalho acontece em workers, não na requisição.
  const emAndamento = (audiencias ?? []).some((a) => STATUS_AUDIENCIA[a.status]?.emAndamento)
  usePolling(() => void recarregar(true), emAndamento)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Audiências</h1>
          <p className="text-sm text-muted-foreground">
            Envie o áudio; a transcrição e a análise acontecem em segundo plano.
          </p>
        </div>
        <DialogUpload aoEnviar={() => void recarregar(true)} />
      </div>

      {erro && <MensagemErro texto={erro} aoTentarNovamente={() => void recarregar()} />}

      {carregando ? (
        <div className="space-y-3">
          {[0, 1, 2].map((i) => <Skeleton key={i} className="h-20 w-full" />)}
        </div>
      ) : (audiencias ?? []).length === 0 && !erro ? (
        <EstadoVazio
          Icone={FileAudio}
          titulo="Nenhuma audiência ainda"
          descricao="Envie a gravação de uma audiência para receber resumo, sugestão estratégica e pontos críticos."
        />
      ) : (
        <div className="space-y-3">
          {(audiencias ?? []).map((a) => {
            const st = STATUS_AUDIENCIA[a.status] ?? STATUS_AUDIENCIA.uploaded
            return (
              <Link key={a.id} to={`/audiencias/${a.id}`} className="block">
                <Card className="transition-colors hover:border-primary/40">
                  <CardContent className="flex items-center gap-4 p-4">
                    <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                      <FileAudio className="size-5 text-muted-foreground" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium">{a.titulo || a.nome_arquivo}</p>
                      <p className="truncate text-sm text-muted-foreground">
                        {AREAS.find((x) => x.valor === a.area_do_direito)?.rotulo ?? a.area_do_direito}
                        {" · "}
                        {formatarData(a.created_at)}
                      </p>
                    </div>
                    <Badge variant={st.variante} className="shrink-0 gap-1.5">
                      {st.emAndamento && <Loader2 className="size-3 animate-spin" />}
                      {st.rotulo}
                    </Badge>
                  </CardContent>
                </Card>
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}

function DialogUpload({ aoEnviar }: { aoEnviar: () => void }) {
  const [aberto, setAberto] = useState(false)
  const [enviando, setEnviando] = useState(false)
  const [arquivo, setArquivo] = useState<File | null>(null)
  const [area, setArea] = useState<AreaDoDireito>("civel")
  const [titulo, setTitulo] = useState("")

  async function submeter(e: React.FormEvent) {
    e.preventDefault()
    if (!arquivo) return
    setEnviando(true)
    try {
      await api.enviarAudiencia(arquivo, area, titulo)
      toast.success("Áudio enviado", {
        description: "A transcrição começou; o status atualiza sozinho.",
      })
      setAberto(false)
      setArquivo(null)
      setTitulo("")
      aoEnviar()
    } catch (err) {
      toast.error("Não foi possível enviar", {
        description: err instanceof Error ? err.message : undefined,
      })
    } finally {
      setEnviando(false)
    }
  }

  return (
    <Dialog open={aberto} onOpenChange={setAberto}>
      <DialogTrigger asChild>
        <Button className="gap-2">
          <Upload className="size-4" />
          Nova audiência
        </Button>
      </DialogTrigger>
      <DialogContent>
        <form onSubmit={submeter}>
          <DialogHeader>
            <DialogTitle>Nova audiência</DialogTitle>
            <DialogDescription>
              O arquivo vai para armazenamento privado; o link de escuta é assinado e de vida curta.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="audio">Áudio da audiência</Label>
              <Input
                id="audio"
                type="file"
                accept="audio/*,video/mp4,.m4a,.mp3,.wav,.ogg"
                required
                onChange={(e) => setArquivo(e.target.files?.[0] ?? null)}
              />
              <p className="text-xs text-muted-foreground">Até 300 MB.</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="titulo">Título</Label>
              <Input
                id="titulo"
                value={titulo}
                onChange={(e) => setTitulo(e.target.value)}
                placeholder="Opcional — usa o nome do arquivo"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="area">Área do direito</Label>
              <Select value={area} onValueChange={(v) => setArea(v as AreaDoDireito)}>
                <SelectTrigger id="area" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {AREAS.map((a) => (
                    <SelectItem key={a.valor} value={a.valor}>{a.rotulo}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                Ancora a análise na norma aplicável.
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button type="submit" disabled={enviando || !arquivo} className="gap-2">
              {enviando && <Loader2 className="size-4 animate-spin" />}
              {enviando ? "Enviando…" : "Enviar"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
