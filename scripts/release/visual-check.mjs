// Visual check of the Codixai UI after the readability pass.
//
// It loads the built frontend (served by the Vite preview server on
// http://127.0.0.1:5180), stubs the Wails runtime so the screens render
// without a window, and takes a screenshot of the audit tab and the
// verification tab. It also switches the language to Spanish and the theme
// to dark to prove the two switches work.
//
// Usage: node scripts/release/visual-check.mjs <screenshot-dir>

import { chromium } from '@playwright/mcp'
import { mkdirSync } from 'node:fs'

const outDir = process.argv[2] || '/tmp/visual'
mkdirSync(outDir, { recursive: true })

const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1280, height: 800 } })

// Stub the Wails runtime before the page loads.
await page.addInitScript(() => {
  window.go = {
    ui: {
      App: {
        Audit: async () => ({ document: { findings: [], summary: { total: 0 } } }),
        AuditAsync: async () => ({ run_id: 'stub', kind: 'progress', progress: { phase: 'validating' } }),
        CancelAudit: async () => {},
        ChooseRepository: async () => '',
        ChooseFile: async () => '',
        Export: async () => '',
        Root: async () => '',
        Verify: async () => ({ document: { results: [] } }),
        VerifyAsync: async () => ({ run_id: 'stub', kind: 'progress', progress: { phase: 'validating' } }),
        Visualize: async () => ({ symbols: [], dependencies: [] }),
      },
    },
  }
  window.runtime = {
    EventsOn: () => () => {},
    EventsEmit: () => {},
    Quit: () => {},
  }
})

await page.goto('http://127.0.0.1:5180/', { waitUntil: 'networkidle' })
await page.waitForSelector('h1', { timeout: 15000 })

// Screenshot 1: audit tab, default state.
await page.screenshot({ path: `${outDir}/01-audit-default.png`, fullPage: false })

// Screenshot 2: audit tab with a stubbed error.
await page.evaluate(() => {
  const app = document.getElementById('app')
  if (!app) return
  // Trigger the bridge-unavailable notice by clearing the stub.
  delete window.go
})
await page.reload({ waitUntil: 'networkidle' })
await page.waitForSelector('h1', { timeout: 15000 })
await page.screenshot({ path: `${outDir}/02-audit-noruntime.png`, fullPage: false })

// Screenshot 3: verification tab.
await page.addInitScript(() => {
  window.go = {
    ui: {
      App: {
        Audit: async () => ({ document: { findings: [], summary: { total: 0 } } }),
        AuditAsync: async () => ({ run_id: 'stub', kind: 'progress', progress: { phase: 'validating' } }),
        CancelAudit: async () => {},
        ChooseRepository: async () => '',
        ChooseFile: async () => '',
        Export: async () => '',
        Root: async () => '',
        Verify: async () => ({ document: { results: [] } }),
        VerifyAsync: async () => ({ run_id: 'stub', kind: 'progress', progress: { phase: 'validating' } }),
        Visualize: async () => ({ symbols: [], dependencies: [] }),
      },
    },
  }
  window.runtime = { EventsOn: () => () => {}, EventsEmit: () => {}, Quit: () => {} }
})
await page.goto('http://127.0.0.1:5180/', { waitUntil: 'networkidle' })
await page.waitForSelector('h1', { timeout: 15000 })
await page.click('a[role="tab"]:has-text("Verification")')
await page.screenshot({ path: `${outDir}/03-verify-tab.png`, fullPage: false })

// Screenshot 4: Spanish language.
await page.click('button[aria-label="Español"]')
await page.screenshot({ path: `${outDir}/04-spanish.png`, fullPage: false })

// Screenshot 5: dark theme.
await page.click('button[aria-label="Tema oscuro"]')
await page.screenshot({ path: `${outDir}/05-dark.png`, fullPage: false })

await browser.close()
console.log(`screenshots written to ${outDir}`)