import type {
  Audiencia,
  AreaDoDireito,
  Movimentacao,
  Notificacao,
  PreferenciasNotificacao,
  Processo,
  Sessao,
} from "./tipos"

// Tudo passa pelo gateway na mesma origem: em dev o proxy do Vite encaminha
// /api para :8080, em produção o nginx serve os estáticos e faz o proxy. O
// código não distingue os dois casos.
const BASE = "/api"

const CHAVE_TOKEN = "meirinho.token"

export function lerToken(): string | null {
  return localStorage.getItem(CHAVE_TOKEN)
}
export function gravarToken(t: string) {
  localStorage.setItem(CHAVE_TOKEN, t)
}
export function limparToken() {
  localStorage.removeItem(CHAVE_TOKEN)
}

/** Erro da API já traduzido: `message` do backend é escrito para o usuário final. */
export class ErroAPI extends Error {
  readonly status: number
  readonly codigo: string

  constructor(status: number, codigo: string, mensagem: string) {
    super(mensagem)
    this.status = status
    this.codigo = codigo
  }
}

/** Sinaliza que o token morreu — o AuthProvider escuta e derruba a sessão. */
export const EVENTO_NAO_AUTORIZADO = "meirinho:nao-autorizado"

async function requisitar<T>(
  caminho: string,
  init: RequestInit = {},
  corpoMultipart = false,
): Promise<T> {
  const headers = new Headers(init.headers)
  if (!corpoMultipart && init.body) {
    headers.set("Content-Type", "application/json")
  }
  const token = lerToken()
  if (token) headers.set("Authorization", `Bearer ${token}`)

  const resp = await fetch(BASE + caminho, { ...init, headers })

  if (resp.status === 401) {
    limparToken()
    window.dispatchEvent(new Event(EVENTO_NAO_AUTORIZADO))
    throw new ErroAPI(401, "nao_autorizado", "Sua sessão expirou. Entre novamente.")
  }

  if (!resp.ok) {
    // O backend responde { error, message } em todo caminho de erro; se algo
    // fora dele vazar (nginx, por exemplo), não deixa a tela sem explicação.
    let codigo = "erro_desconhecido"
    let mensagem = `Falha na requisição (HTTP ${resp.status}).`
    try {
      const corpo = await resp.json()
      codigo = corpo.error ?? codigo
      if (corpo.message) mensagem = corpo.message
    } catch {
      /* resposta sem JSON — mantém a mensagem genérica */
    }
    throw new ErroAPI(resp.status, codigo, mensagem)
  }

  if (resp.status === 204) return undefined as T
  return (await resp.json()) as T
}

export const api = {
  cadastro: (dados: { nome: string; oab: string; email: string; senha: string }) =>
    requisitar<Sessao>("/auth/cadastro", { method: "POST", body: JSON.stringify(dados) }),

  login: (dados: { email: string; senha: string }) =>
    requisitar<Sessao>("/auth/login", { method: "POST", body: JSON.stringify(dados) }),

  eu: () => requisitar<Sessao["tenant"]>("/auth/eu"),

  atualizarPerfil: (dados: { nome: string; oab: string; email: string }) =>
    requisitar<Sessao["tenant"]>("/auth/eu", { method: "PATCH", body: JSON.stringify(dados) }),

  trocarSenha: (dados: { senha_atual: string; nova_senha: string }) =>
    requisitar<void>("/auth/senha", { method: "POST", body: JSON.stringify(dados) }),

  preferencias: () => requisitar<PreferenciasNotificacao>("/notifications/preferencias"),

  salvarPreferencias: (p: PreferenciasNotificacao) =>
    requisitar<PreferenciasNotificacao>("/notifications/preferencias", {
      method: "PUT",
      body: JSON.stringify(p),
    }),

  audiencias: () =>
    requisitar<{ itens: Audiencia[] }>("/hearings").then((r) => r.itens ?? []),

  audiencia: (id: string) => requisitar<Audiencia>(`/hearings/${id}`),

  enviarAudiencia: (arquivo: File, area: AreaDoDireito, titulo: string) => {
    const form = new FormData()
    form.append("audio", arquivo)
    form.append("area_do_direito", area)
    form.append("titulo", titulo)
    return requisitar<Audiencia>("/hearings", { method: "POST", body: form }, true)
  },

  processos: () =>
    requisitar<{ itens: Processo[] }>("/processes").then((r) => r.itens ?? []),

  processo: (id: string) => requisitar<Processo>(`/processes/${id}`),

  processoDaMovimentacao: (movimentacaoID: string) =>
    requisitar<{ process_id: string }>(`/processes/movimentacoes/${movimentacaoID}/processo`).then(
      (r) => r.process_id,
    ),

  criarProcesso: (dados: {
    numero_cnj: string
    titulo: string
    tribunal: string
    area_do_direito: AreaDoDireito
  }) => requisitar<Processo>("/processes", { method: "POST", body: JSON.stringify(dados) }),

  arquivarProcesso: (id: string) =>
    requisitar<{ status: string }>(`/processes/${id}`, { method: "DELETE" }),

  movimentacoes: (processoID: string) =>
    requisitar<{ itens: Movimentacao[] }>(`/processes/${processoID}/movements`).then(
      (r) => r.itens ?? [],
    ),

  notificacoes: (apenasNaoLidas = false) =>
    requisitar<{ itens: Notificacao[] }>(
      `/notifications${apenasNaoLidas ? "?nao_lidas=true" : ""}`,
    ).then((r) => r.itens ?? []),

  resumoNotificacoes: () => requisitar<{ nao_lidas: number }>("/notifications/resumo"),

  marcarLida: (id: string) =>
    requisitar<{ lida: boolean }>(`/notifications/${id}/lida`, { method: "POST" }),

  marcarTodasLidas: () =>
    requisitar<{ marcadas: number }>("/notifications/lidas", { method: "POST" }),
}
