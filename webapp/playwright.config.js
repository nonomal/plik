import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
    testDir: './e2e',
    fullyParallel: false,
    forbidOnly: !!process.env.CI,
    retries: process.env.CI ? 1 : 0,
    workers: 1,
    reporter: process.env.CI ? 'github' : 'list',

    use: {
        baseURL: process.env.BASE_URL || 'http://localhost:8585',
        screenshot: 'only-on-failure',
        trace: 'retain-on-failure',
    },

    projects: [
        {
            name: 'chromium',
            use: { ...devices['Desktop Chrome'] },
            testIgnore: /subpath\.spec/,
        },
        {
            name: 'chromium-subpath',
            use: {
                ...devices['Desktop Chrome'],
                baseURL: 'http://localhost:8586/sub/',
            },
            testMatch: /subpath\.spec/,
        },
    ],

    webServer: [
        {
            command: 'bash e2e/start-server.sh',
            url: 'http://localhost:8585/version',
            reuseExistingServer: !process.env.CI,
            timeout: 30_000,
            stdout: 'pipe',
            stderr: 'pipe',
        },
        {
            command: 'bash e2e/start-server-subpath.sh',
            url: 'http://localhost:8586/sub/version',
            reuseExistingServer: !process.env.CI,
            timeout: 30_000,
            stdout: 'pipe',
            stderr: 'pipe',
        },
    ],

    globalTeardown: './e2e/global-teardown.js',
})
