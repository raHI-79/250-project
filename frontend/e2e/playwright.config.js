import { defineConfig, devices } from '@playwright/test'

const BASE_URL = process.env.E2E_BASE_URL || 'http://localhost:5173'

const PHONE_SPEC = /phones\.spec\.js/

const PHONES = [
  { name: 'phone-web-390', width: 390, height: 844, appShell: false },
  { name: 'phone-web-360', width: 360, height: 740, appShell: false },
  { name: 'phone-app-390', width: 390, height: 844, appShell: true },
  { name: 'phone-app-360', width: 360, height: 740, appShell: true },
]

export default defineConfig({
  testDir: './specs',
  globalSetup: './setup/grant.js',
  outputDir: './.artifacts',
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
      testIgnore: PHONE_SPEC,
      use: { ...devices['Pixel 5'], viewport: { width: 390, height: 844 }, colorScheme: 'light' },
    },
    {
      name: 'phone-dark',
      testIgnore: PHONE_SPEC,
      use: { ...devices['Pixel 5'], viewport: { width: 390, height: 844 }, colorScheme: 'dark' },
    },
    {
      name: 'desktop-light',
      testIgnore: PHONE_SPEC,
      use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 800 }, colorScheme: 'light' },
    },
    {
      name: 'desktop-dark',
      testIgnore: PHONE_SPEC,
      use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 800 }, colorScheme: 'dark' },
    },
    ...PHONES.map((phone) => ({
      name: phone.name,
      testMatch: PHONE_SPEC,
      use: {
        ...devices['Pixel 5'],
        viewport: { width: phone.width, height: phone.height },
        colorScheme: 'light',
        appShell: phone.appShell,
      },
    })),
  ],
})
