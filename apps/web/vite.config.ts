import path from "path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  server: {
    // Em dev o front fala com o gateway do compose; em produção o nginx serve
    // os estáticos e o /api cai no mesmo host, então o caminho é idêntico nos
    // dois casos e o código nunca precisa saber em qual está.
    proxy: {
      "/api": { target: "http://localhost:8080", changeOrigin: true },
    },
  },
})
