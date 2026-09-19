import { cn } from "@/lib/utils"

/**
 * Marca do Meirinho: o frontão de um fórum cujas colunas desenham o "M".
 * Desenhada à mão em grade de 32 — sem contêiner em volta, para funcionar
 * sobre fundo claro e escuro com `currentColor`.
 */
export function MarcaMeirinho({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" fill="currentColor" aria-hidden="true" className={cn("size-8", className)}>
      {/* frontão */}
      <path d="M16 3.5 29 11H3z" />
      {/* arquitrave */}
      <rect x="3" y="12.5" width="26" height="2.2" />
      {/* colunas em M */}
      <rect x="5.5" y="16.5" width="3.4" height="9.5" />
      <rect x="23.1" y="16.5" width="3.4" height="9.5" />
      <path d="M8.9 16.5h3.3L16 21.3l3.8-4.8h3.3L16 25.4z" />
      {/* base */}
      <rect x="3" y="27.5" width="26" height="2.2" />
    </svg>
  )
}

/** Marca + nome, para cabeçalhos. */
export function LogoMeirinho({ recolhido = false, className }: { recolhido?: boolean; className?: string }) {
  return (
    <span className={cn("flex items-center gap-2.5 text-foreground", className)}>
      <MarcaMeirinho className="size-7 shrink-0" />
      {!recolhido && (
        <span className="font-logo text-[26px] leading-none tracking-[-0.01em]">Meirinho</span>
      )}
    </span>
  )
}
