import { defineConfig } from '@playwright/test'

const baseURL = process.env.PW_BASE_URL || 'http://127.0.0.1:5176'

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  expect: { timeout: 30_000 },
  fullyParallel: false,
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure'
  },
  reporter: [['list'], ['html', { open: 'never' }]]
})
