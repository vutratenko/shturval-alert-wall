import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.ts',
      injectRegister: false,
      manifest: {
        id: '/',
        name: 'Штурвал · Пульт алертов',
        short_name: 'Пульт',
        description: 'Живая стена firing-алертов кластеров Штурвала',
        lang: 'ru',
        start_url: '/wall',
        scope: '/',
        display: 'standalone',
        background_color: '#000000',
        theme_color: '#000000',
        icons: [
          { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
          { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
          { src: '/icons/icon-192-maskable.png', sizes: '192x192', type: 'image/png', purpose: 'maskable' },
          { src: '/icons/icon-512-maskable.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
      injectManifest: {
        globPatterns: ['**/*.{js,css,html,svg,jpg,png,woff2,webmanifest}'],
      },
      devOptions: { enabled: false },
    }),
  ],
  build: { outDir: '../cmd/server/static', emptyOutDir: true },
  publicDir: 'public',
});
