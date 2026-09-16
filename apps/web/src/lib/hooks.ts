import { useCallback, useEffect, useRef, useState } from "react"

/**
 * Carrega dado de forma assíncrona expondo os três estados que toda tela
 * precisa (carregando / erro / dado) mais um recarregar manual.
 */
export function useDados<T>(carregar: () => Promise<T>, deps: unknown[] = []) {
  const [dado, setDado] = useState<T | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const [carregando, setCarregando] = useState(true)

  const fn = useRef(carregar)
  fn.current = carregar

  const recarregar = useCallback(async (silencioso = false) => {
    if (!silencioso) setCarregando(true)
    try {
      setDado(await fn.current())
      setErro(null)
    } catch (e) {
      setErro(e instanceof Error ? e.message : "Erro inesperado.")
    } finally {
      setCarregando(false)
    }
  }, [])

  useEffect(() => {
    void recarregar()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  return { dado, erro, carregando, recarregar }
}

/**
 * Repete uma ação enquanto `ativo` for verdadeiro.
 *
 * O pipeline é assíncrono: a audiência sai do upload em `uploaded` e só vira
 * `analyzed` depois de passar por dois workers. Sem isso a tela ficaria
 * mostrando "Na fila" indefinidamente e só a tecla F5 contaria a verdade.
 */
export function usePolling(acao: () => void, ativo: boolean, intervaloMs = 3000) {
  const fn = useRef(acao)
  fn.current = acao

  useEffect(() => {
    if (!ativo) return
    const t = setInterval(() => fn.current(), intervaloMs)
    return () => clearInterval(t)
  }, [ativo, intervaloMs])
}
