import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { TanStackRouterVite } from '@tanstack/router-plugin/vite';
import path from 'path';

function getPackageName(id: string): string | null {
  const normalized = id.replace(/^\0/, '').split('?')[0].replace(/\\/g, '/');
  const marker = '/node_modules/';
  const idx = normalized.lastIndexOf(marker);
  if (idx === -1) return null;
  const rest = normalized.slice(idx + marker.length);
  const parts = rest.split('/');
  if (parts.length === 0 || !parts[0]) return null;
  return parts[0].startsWith('@') && parts.length > 1
    ? `${parts[0]}/${parts[1]}`
    : parts[0];
}

export default defineConfig({
  plugins: [
    TanStackRouterVite({
      routesDirectory: './src/routes',
      generatedRouteTree: './src/routeTree.gen.ts',
    }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },
  build: {
    target: 'esnext',
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id === '\0commonjsHelpers.js') {
            return 'vendor-react';
          }

          const pkg = getPackageName(id);
          if (!pkg) return undefined;

          if (
            pkg === 'react' ||
            pkg === 'react-dom' ||
            pkg === 'scheduler' ||
            pkg === 'use-sync-external-store' ||
            pkg === 'tslib' ||
            pkg === '@babel/runtime'
          ) {
            return 'vendor-react';
          }

          if (pkg.startsWith('@tanstack/') || pkg === 'seroval') {
            return 'vendor-tanstack';
          }

          if (
            pkg.startsWith('@radix-ui/') ||
            pkg.startsWith('@floating-ui/') ||
            pkg === 'aria-hidden' ||
            pkg === 'react-remove-scroll' ||
            pkg === 'react-remove-scroll-bar' ||
            pkg === 'react-style-singleton' ||
            pkg === 'get-nonce' ||
            pkg === 'use-callback-ref' ||
            pkg === 'use-sidecar' ||
            pkg === 'detect-node-es'
          ) {
            return 'vendor-radix';
          }

          if (
            pkg === 'i18next' ||
            pkg === 'react-i18next' ||
            pkg === 'html-parse-stringify' ||
            pkg === 'void-elements'
          ) {
            return 'vendor-i18n';
          }

          if (
            pkg === 'motion' ||
            pkg === 'framer-motion' ||
            pkg === 'motion-dom' ||
            pkg === 'motion-utils'
          ) {
            return 'vendor-motion';
          }

          if (
            pkg === 'recharts' ||
            pkg === 'recharts-scale' ||
            pkg === 'react-smooth' ||
            pkg === 'react-is' ||
            pkg === 'react-transition-group' ||
            pkg === 'dom-helpers' ||
            pkg === 'prop-types' ||
            pkg === 'fast-equals' ||
            pkg === 'object-assign' ||
            pkg === 'victory-vendor' ||
            pkg === 'eventemitter3' ||
            pkg === 'tiny-invariant' ||
            pkg === 'decimal.js-light' ||
            pkg === 'internmap' ||
            pkg === 'lodash' ||
            pkg.startsWith('d3-')
          ) {
            return 'vendor-charts';
          }

          if (
            pkg === 'lucide-react' ||
            pkg === 'clsx' ||
            pkg === 'tailwind-merge' ||
            pkg === 'sonner' ||
            pkg === 'cmdk'
          ) {
            return 'vendor-ui';
          }

          return undefined;
        },
      },
    },
  },
});

