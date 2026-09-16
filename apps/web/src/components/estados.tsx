import type { LucideIcon } from "lucide-react"
import { AlertCircle } from "lucide-react"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"

/** Estado vazio: a lista está certa, só não tem nada nela ainda. */
export function EstadoVazio({
  Icone, titulo, descricao, acao,
}: {
  Icone: LucideIcon
  titulo: string
  descricao: string
  acao?: React.ReactNode
}) {
  return (
    <Card className="border-dashed">
      <CardContent className="flex flex-col items-center gap-3 px-6 py-14 text-center">
        <div className="flex size-11 items-center justify-center rounded-xl bg-muted">
          <Icone className="size-5 text-muted-foreground" />
        </div>
        <div className="space-y-1">
          <p className="font-medium">{titulo}</p>
          <p className="mx-auto max-w-sm text-sm text-muted-foreground">{descricao}</p>
        </div>
        {acao}
      </CardContent>
    </Card>
  )
}

/** Erro de carregamento, com a saída óbvia: tentar de novo. */
export function MensagemErro({
  texto, aoTentarNovamente,
}: {
  texto: string
  aoTentarNovamente?: () => void
}) {
  return (
    <Alert variant="destructive">
      <AlertCircle className="size-4" />
      <AlertDescription className="flex items-center justify-between gap-4">
        <span>{texto}</span>
        {aoTentarNovamente && (
          <Button variant="outline" size="sm" onClick={aoTentarNovamente}>
            Tentar novamente
          </Button>
        )}
      </AlertDescription>
    </Alert>
  )
}
