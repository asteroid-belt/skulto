import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// `base` controls the public path the built assets are served from.
// On GitHub Pages project sites the app lives under /<repo-name>/, so CI
// passes that subpath via the WT_BASE env var. Defaults to '/' for local dev.
export default defineConfig({
  base: process.env.WT_BASE || '/',
  plugins: [react()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
});
