import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// En développement, l'API Go tourne sur 127.0.0.1:18890 (voir docs/DEVELOPPEMENT.md).
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: { '/api': { target: process.env.AURORA_API ?? 'http://127.0.0.1:18890', changeOrigin: false } },
  },
  build: {
    // Sortie directement dans le dossier embarqué par le binaire Go. Jamais vidé par Vite :
    // scripts/build.sh met l'ancien contenu à la Corbeille (aucune suppression).
    outDir: '../internal/webui/dist', emptyOutDir: false,
    target: ['safari15', 'chrome100', 'firefox100', 'edge100'], sourcemap: false, chunkSizeWarningLimit: 900 },
  test: { environment: 'jsdom' },
})
