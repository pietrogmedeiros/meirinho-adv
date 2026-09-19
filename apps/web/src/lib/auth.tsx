import { createContext, useCallback, useContext, useEffect, useState } from "react"
import type { ReactNode } from "react"
import { EVENTO_NAO_AUTORIZADO, api, gravarToken, lerToken, limparToken } from "./api"
import type { Tenant } from "./tipos"

interface ContextoAuth {
  tenant: Tenant | null
  carregando: boolean
  entrar: (email: string, senha: string) => Promise<void>
  cadastrar: (d: { nome: string; oab: string; email: string; senha: string }) => Promise<void>
  sair: () => void
  /** Troca os dados da sessão depois de editar o perfil. */
  atualizarTenant: (t: Tenant) => void
}

const Ctx = createContext<ContextoAuth | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [tenant, setTenant] = useState<Tenant | null>(null)
  // Começa carregando se há token: a tela não deve piscar o login enquanto a
  // sessão guardada ainda está sendo validada contra o servidor.
  const [carregando, setCarregando] = useState(() => lerToken() !== null)

  const sair = useCallback(() => {
    limparToken()
    setTenant(null)
  }, [])

  useEffect(() => {
    if (!lerToken()) return
    // Um token no localStorage não prova nada: pode estar expirado ou ter sido
    // emitido por outro ambiente. Quem decide é o /auth/eu.
    api
      .eu()
      .then(setTenant)
      .catch(() => limparToken())
      .finally(() => setCarregando(false))
  }, [])

  // Qualquer 401 em qualquer requisição derruba a sessão, venha de onde vier.
  useEffect(() => {
    window.addEventListener(EVENTO_NAO_AUTORIZADO, sair)
    return () => window.removeEventListener(EVENTO_NAO_AUTORIZADO, sair)
  }, [sair])

  const entrar = useCallback(async (email: string, senha: string) => {
    const s = await api.login({ email, senha })
    gravarToken(s.token)
    setTenant(s.tenant)
  }, [])

  const cadastrar = useCallback(
    async (d: { nome: string; oab: string; email: string; senha: string }) => {
      const s = await api.cadastro(d)
      gravarToken(s.token)
      setTenant(s.tenant)
    },
    [],
  )

  return (
    <Ctx.Provider value={{ tenant, carregando, entrar, cadastrar, sair, atualizarTenant: setTenant }}>
      {children}
    </Ctx.Provider>
  )
}

export function useAuth() {
  const c = useContext(Ctx)
  if (!c) throw new Error("useAuth precisa estar dentro de AuthProvider")
  return c
}
