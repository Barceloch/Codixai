import { defineConfig, type Plugin } from 'vite'
import { writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

// keepDistPlaceholder restores the placeholder that keeps the embedded
// directory in the repository.
//
// Vite empties dist before every build, which deletes the committed
// .gitkeep. The file has to exist after the build too, because the Go
// package frontend embeds dist and the repository must compile before the
// first build of the frontend.
function keepDistPlaceholder(): Plugin {
  return {
    name: 'codixai-dist-placeholder',
    closeBundle() {
      writeFileSync(
        fileURLToPath(new URL('./dist/.gitkeep', import.meta.url)),
        '',
      )
    },
  }
}

// The frontend is served by the Wails v2 asset server in production and by
// the Vite dev server during development. The build output is frontend/dist,
// which the frontend Go package embeds into the binary.
export default defineConfig({
  plugins: [tailwindcss(), svelte(), keepDistPlaceholder()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    strictPort: true,
  },
})
