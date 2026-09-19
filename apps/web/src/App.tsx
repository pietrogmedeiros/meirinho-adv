import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { Toaster } from "@/components/ui/sonner"
import { Layout } from "@/components/layout"
import { AuthProvider, useAuth } from "@/lib/auth"
import { AudienciaDetalhePage } from "@/pages/audiencia-detalhe"
import { AudienciasPage } from "@/pages/audiencias"
import { ContaPage } from "@/pages/conta"
import { InicioPage } from "@/pages/inicio"
import { LoginPage } from "@/pages/login"
import { NotificacoesPage } from "@/pages/notificacoes"
import { ProcessoDetalhePage } from "@/pages/processo-detalhe"
import { ProcessosPage } from "@/pages/processos"

function Rotas() {
  const { tenant, carregando } = useAuth()

  if (carregando) {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // Sem sessão, qualquer caminho cai no login; ao entrar, o estado muda e a
  // mesma URL passa a renderizar a tela pedida — o link profundo sobrevive.
  if (!tenant) return <LoginPage />

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<InicioPage />} />
        <Route path="/audiencias" element={<AudienciasPage />} />
        <Route path="/audiencias/:id" element={<AudienciaDetalhePage />} />
        <Route path="/processos" element={<ProcessosPage />} />
        <Route path="/processos/:id" element={<ProcessoDetalhePage />} />
        <Route path="/notificacoes" element={<NotificacoesPage />} />
        <Route path="/conta" element={<ContaPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Rotas />
        <Toaster position="top-right" richColors />
      </AuthProvider>
    </BrowserRouter>
  )
}
