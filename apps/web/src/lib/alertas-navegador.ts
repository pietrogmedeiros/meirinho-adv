// Alerta nativo do navegador quando chega notificação nova. É preferência do
// aparelho, não da conta (a permissão é concedida por navegador), então fica
// no localStorage e não no servidor.

const CHAVE = "meirinho.alertas-navegador"

export function suportaAlertas(): boolean {
  return typeof window !== "undefined" && "Notification" in window
}

export function alertasAtivos(): boolean {
  if (!suportaAlertas() || Notification.permission !== "granted") return false
  try {
    return localStorage.getItem(CHAVE) === "1"
  } catch {
    return false
  }
}

/** Pede a permissão (se preciso) e liga. Devolve se ficou ligado. */
export async function ativarAlertas(): Promise<boolean> {
  if (!suportaAlertas()) return false
  const permissao =
    Notification.permission === "default" ? await Notification.requestPermission() : Notification.permission
  const ok = permissao === "granted"
  try {
    localStorage.setItem(CHAVE, ok ? "1" : "0")
  } catch {
    /* sem storage, vale só para esta aba */
  }
  return ok
}

export function desativarAlertas() {
  try {
    localStorage.setItem(CHAVE, "0")
  } catch {
    /* idem */
  }
}

export function mostrarAlerta(titulo: string, corpo: string, aoClicar?: () => void) {
  if (!alertasAtivos()) return
  const n = new Notification(titulo, { body: corpo, icon: "/favicon.svg", tag: titulo })
  n.onclick = () => {
    window.focus()
    aoClicar?.()
    n.close()
  }
}
