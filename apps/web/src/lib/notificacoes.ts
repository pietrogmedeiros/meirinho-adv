import { api } from "./api"
import type { Notificacao } from "./tipos"

/**
 * Para onde a notificação leva. A de audiência referencia a própria audiência;
 * a de movimentação referencia a movimentação, então é preciso perguntar ao
 * process-monitor de qual processo ela é. Se falhar, cai na carteira.
 */
export async function rotaDaNotificacao(n: Notificacao): Promise<string | null> {
  if (n.categoria === "audiencia" && n.referencia_id) return `/audiencias/${n.referencia_id}`
  if (n.categoria === "movimentacao") {
    if (!n.referencia_id) return "/processos"
    try {
      return `/processos/${await api.processoDaMovimentacao(n.referencia_id)}`
    } catch {
      return "/processos"
    }
  }
  return null
}
