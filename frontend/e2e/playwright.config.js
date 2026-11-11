import { defineConfig, devices } from '@playwright/test'

const BASE_URL = process.env.E2E_BASE_URL || 'http://localhost:5173'

// The suite drives a real browser against a local backend with the demo seed, so it runs
// on its own and never in the same job as the unit tests.
export default defineConfig({
  testDir: './specs',
  globalSetup: './setup/grant.js',
  outputDir: './.artifacts',
  // A role's walk opens a dozen pages and measures each one, so a case is minutes
  // rather than seconds.
  timeout: 360_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['list']] : [['list']],
  use: {
    baseURL: BASE_URL,
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
    trace: 'off',
    video: 'off',
    screenshot: 'off',
  },
  projects: [
    {
      name: 'phone-light',
      use: { ...devices['Pixel 5'], viewport: { width: 390, height: 844 }, colorScheme: 'light' },
    },
    {
      name: 'phone-dark',
      use: { ...devices['Pixel 5'], viewport: { width: 390, height: 844 }, colorScheme: 'dark' },
    },
    {
      name: 'desktop-light',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 800 }, colorScheme: 'light' },
    },
    {
      name: 'desktop-dark',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 800 }, colorScheme: 'dark' },
    },
  ],
})
