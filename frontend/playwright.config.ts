import { defineConfig, devices } from '@playwright/test'

// End-to-end tests run against the real single binary (make build), serving
// the embedded interface with a throwaway database under .e2e/.
const port = 8095
const baseURL = `http://127.0.0.1:${port}`

export default defineConfig({
  testDir: 'e2e',
  testMatch: '**/*.e2e.ts',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: { baseURL, trace: 'retain-on-failure' },
  projects: [
    { name: 'setup', testMatch: 'setup.e2e.ts' },
    {
      name: 'desktop',
      use: { ...devices['Desktop Chrome'] },
      dependencies: ['setup'],
      testIgnore: ['setup.e2e.ts', 'layout.e2e.ts'],
    },
    {
      // AGENTS.md: every page must work at 390px wide, in both themes.
      name: 'mobile',
      use: { ...devices['Desktop Chrome'], viewport: { width: 390, height: 844 } },
      dependencies: ['setup'],
      testMatch: 'layout.e2e.ts',
    },
  ],
  webServer: {
    command: `rm -rf .e2e && ../bin/repo-scout serve --addr 127.0.0.1:${port} --db .e2e/reposcout.db`,
    url: `${baseURL}/api/health`,
    reuseExistingServer: false,
    timeout: 30_000,
  },
})
