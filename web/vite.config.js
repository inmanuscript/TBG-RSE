import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [
    vue(),
    {
      name: 'cache-bust-assets',
      transformIndexHtml(html) {
        const v = Date.now()
        return html
          .replace('/assets/index.js', `/assets/index.js?v=${v}`)
          .replace('/assets/index.css', `/assets/index.css?v=${v}`)
      },
    },
  ],
  build: {
    outDir: '../webui/dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        entryFileNames: 'assets/index.js',
        chunkFileNames: 'assets/[name].js',
        assetFileNames: (assetInfo) => {
          const name = assetInfo.names?.[0] ?? assetInfo.name ?? ''
          if (name.endsWith('.css')) return 'assets/index.css'
          return 'assets/[name].[ext]'
        },
      },
    },
  },
  server: {
    proxy: {
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})
