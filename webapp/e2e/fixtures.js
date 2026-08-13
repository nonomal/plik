import { test as base, expect } from '@playwright/test'

/** Admin credentials used by start-server.sh */
export const ADMIN_LOGIN = 'admin'
export const ADMIN_PASSWORD = 'plikplik'

/**
 * Extended test fixtures for Plik e2e tests.
 *
 * - `authenticatedPage`: a Page that has already logged in as admin.
 * - `withConfig`: patches /config API responses with arbitrary overrides.
 */
export const test = base.extend({
    /**
     * Patches the /config API response with arbitrary overrides.
     * Must be called BEFORE page.goto() so the init script is active
     * when the SPA boots.
     *
     * Usage:
     *   test('my test', async ({ page, withConfig }) => {
     *       await withConfig({ feature_authentication: 'forced' })
     *       await page.goto('/')
     *   })
     */
    withConfig: async ({ page }, use) => {
        async function withConfig(overrides) {
            await page.addInitScript((ov) => {
                const origFetch = window.fetch
                window.fetch = async function (...args) {
                    const response = await origFetch.apply(this, args)
                    const url = typeof args[0] === 'string' ? args[0] : args[0]?.url || ''
                    if (url.endsWith('/config')) {
                        const json = await response.json()
                        Object.assign(json, ov)
                        return new Response(JSON.stringify(json), {
                            status: response.status,
                            statusText: response.statusText,
                            headers: response.headers,
                        })
                    }
                    return response
                }
            }, overrides)
        }
        await use(withConfig)
    },

    /**
     * Patches the /version API response with arbitrary overrides.
     * Uses page.route() to intercept API calls reliably, even during
     * SPA hash-based navigation where addInitScript won't re-run.
     *
     * Usage:
     *   test('my test', async ({ authenticatedPage, withVersion }) => {
     *       await withVersion({ isRelease: true, isMint: false })
     *       await authenticatedPage.goto('/#/admin')
     *   })
     */
    withVersion: async ({ page }, use) => {
        async function withVersion(overrides) {
            await page.route('**/version', async (route) => {
                const response = await route.fetch()
                const json = await response.json()
                Object.assign(json, overrides)
                await route.fulfill({
                    status: response.status(),
                    headers: response.headers(),
                    body: JSON.stringify(json),
                })
            })
        }
        await use(withVersion)
    },

    /**
     * Intercepts /settings.json with the given overrides.
     * Pass an object to merge with defaults, or `null` to simulate a 404.
     * Must be called BEFORE page.goto().
     *
     * Usage:
     *   test('my test', async ({ page, withSettings }) => {
     *       await withSettings({ name: 'MyApp', theme: 'nord' })
     *       await page.goto('/')
     *   })
     *   // 404:
     *   test('no settings', async ({ page, withSettings }) => {
     *       await withSettings(null)
     *       await page.goto('/')
     *   })
     */
    withSettings: async ({ page }, use) => {
        async function withSettings(overrides) {
            await page.route('**/settings.json', async (route) => {
                if (overrides === null) {
                    await route.fulfill({ status: 404, body: '' })
                } else {
                    await route.fulfill({
                        status: 200,
                        contentType: 'application/json',
                        body: JSON.stringify(overrides),
                    })
                }
            })
        }
        await use(withSettings)
    },

    /**
     * Convenience wrapper: intercepts settings.json with theme picker config.
     * Merges `{ name: 'Plik', theme: 'auto', themes }` with optional extras.
     * Must be called BEFORE page.goto().
     *
     * Usage:
     *   test('my test', async ({ page, withThemes }) => {
     *       await withThemes(['dark', 'light', 'nord'])
     *       await page.goto('/')
     *   })
     *   // With extra settings:
     *   await withThemes(["*"], { defaultDarkTheme: 'solarized-dark' })
     */
    withThemes: async ({ page }, use) => {
        async function withThemes(themes, extra = {}) {
            await page.route('**/settings.json', async (route) => {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        name: 'Plik',
                        theme: 'auto',
                        themes,
                        ...extra,
                    }),
                })
            })
        }
        await use(withThemes)
    },

    /**
     * Convenience wrapper: intercepts settings.json with language picker config.
     * Merges `{ name: 'Plik', language: 'auto', languages }` with optional extras.
     * Must be called BEFORE page.goto().
     *
     * Usage:
     *   test('my test', async ({ page, withLanguages }) => {
     *       await withLanguages(['en', 'fr', 'de'])
     *       await page.goto('/')
     *   })
     */
    withLanguages: async ({ page }, use) => {
        async function withLanguages(languages, extra = {}) {
            await page.route('**/settings.json', async (route) => {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        name: 'Plik',
                        language: 'auto',
                        languages,
                        ...extra,
                    }),
                })
            })
        }
        await use(withLanguages)
    },

    /**
     * Provides a page that is already authenticated as the admin user.
     * Login is done via the API (faster than filling the form).
     */
    authenticatedPage: async ({ page }, use) => {
        // Navigate to the app so the cookie domain is set
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Login via API — must include XSRF token from cookie
        const resp = await page.evaluate(async (creds) => {
            // Read the XSRF token set by the server on first response
            const xsrfMatch = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
            const xsrf = xsrfMatch ? xsrfMatch[1] : ''

            const headers = { 'Content-Type': 'application/json' }
            if (xsrf) headers['X-XSRFToken'] = xsrf

            const r = await fetch('/auth/local/login', {
                method: 'POST',
                credentials: 'same-origin',
                headers,
                body: JSON.stringify(creds),
            })
            return { status: r.status, text: await r.text() }
        }, { login: ADMIN_LOGIN, password: ADMIN_PASSWORD })

        if (resp.status !== 200) {
            throw new Error(`Login failed with status ${resp.status}: ${resp.text}`)
        }

        // Reload so the app picks up the session
        await page.reload({ waitUntil: 'networkidle' })

        await use(page)
    },
})

export { expect }

/**
 * Upload a test file through the UI and return the resulting download URL.
 * Call this on the upload page (/).
 */
export async function uploadTestFile(page, filename = 'test.txt', content = 'hello world') {
    await page.goto('/')
    await page.waitForLoadState('networkidle')

    // Create a file and set it on the hidden file input
    const buffer = Buffer.from(content)
    const input = page.locator('input[type="file"]')
    await input.setInputFiles({
        name: filename,
        mimeType: 'text/plain',
        buffer,
    })

    // Wait for file to appear in pending list, then click the green Upload button
    // (exact: true avoids matching "Create empty upload")
    await page.getByRole('button', { name: 'Upload', exact: true }).click()

    // Wait for navigation to download view (URL will have ?id=)
    await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
    await page.waitForLoadState('networkidle')

    return page.url()
}
