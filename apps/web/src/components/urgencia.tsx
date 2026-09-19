import { Badge } from "@/components/ui/badge"
import { URGENCIA } from "@/lib/format"
import type { Urgencia } from "@/lib/tipos"
import { cn } from "@/lib/utils"

/** Selo de urgência com o ponto colorido — o mesmo em toda tela. */
export function SeloUrgencia({
  urgencia, prefixo = false, className,
}: {
  urgencia: Urgencia
  prefixo?: boolean
  className?: string
}) {
  const u = URGENCIA[urgencia]
  return (
    <Badge variant="outline" className={cn("gap-1.5 bg-background font-medium", className)}>
      <span className={cn("size-1.5 rounded-full", u.cor)} />
      {prefixo ? `Urgência ${u.rotulo.toLowerCase()}` : u.rotulo}
    </Badge>
  )
}
