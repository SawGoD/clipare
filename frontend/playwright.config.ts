import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  timeout: 20_000,
  expect: { timeout: 5_000, toHaveScreenshot: { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.006 } },
  use: { ...devices['Desktop Chrome'], viewport: { width: 700, height: 850 }, deviceScaleFactor: 1, reducedMotion: 'reduce', baseURL: 'http://127.0.0.1:5173' },
  webServer: { command: 'npm run dev', url: 'http://127.0.0.1:5173', reuseExistingServer: false, timeout: 30_000 },
  reporter: [['list']],
})
