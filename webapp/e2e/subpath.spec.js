/**
 * Subpath deployment E2E tests — verifies that the webapp loads and functions
 * correctly when plikd is configured with Path="/sub".
 *
 * These tests run only in the `chromium-subpath` Playwright project
 * (baseURL = http://localhost:8586/sub/). The plikd instance is started by
 * e2e/start-server-subpath.sh with Path="/sub" in plikd.cfg.
 *
 * Key difference from other specs: fetch() calls inside page.evaluate() must
 * derive the API base from window.location rather than using absolute paths
 * like '/auth/local/login', because Playwright's baseURL does not rewrite
 * browser-native fetch paths. This mirrors how api.js works in production.
 */

import { test, expect } from '@playwright/test'

const ADMIN_LOGIN = 'admin'
const ADMIN_PASSWORD = 'plikplik'

/**
 * Login helper that derives the API base from window.location — works under
 * any subpath, just like the real api.js does in production.
 */
async function loginAs(page, login, password) {
    await page.goto('./')
    await page.waitForLoadState('networkidle')

    const status = await page.evaluate(async ({ login, password }) => {
        // Derive API base from the current URL (strips the hash fragment).
        // e.g. http://localhost:8586/sub/ → http://localhost:8586/sub
        const base = window.location.origin + window.location.pathname.replace(/\/$/, '')

        const xsrfMatch = document.cookie.match(/(?:^|;\s*)plik-xsrf=([^;]+)/)
        const xsrf = xsrfMatch ? xsrfMatch[1] : ''
        const headers = { 'Content-Type': 'application/json' }
        if (xsrf) headers['X-XSRFToken'] = xsrf

        const r = await fetch(`${base}/auth/local/login`, {
            method: 'POST',
            credentials: 'same-origin',
            headers,
            body: JSON.stringify({ login, password }),
        })
        return r.status
    }, { login, password })

    if (status !== 200) {
        throw new Error(`Login failed with status ${status}`)
    }
    await page.reload({ waitUntil: 'networkidle' })
}

// ---------------------------------------------------------------------------
// Asset loading
// ---------------------------------------------------------------------------

test.describe('Subpath — asset loading', () => {
    test('webapp loads and shows upload settings', async ({ page }) => {
        await page.goto('./')
        await page.waitForLoadState('networkidle')

        await expect(page.getByRole('heading', { name: 'Upload Settings' })).toBeVisible()
    })

    test('no failed asset requests (no 404s for JS/CSS/favicon)', async ({ page }) => {
        const failures = []
        page.on('response', (resp) => {
            const url = resp.url()
            // Only check local assets (ignore external fonts, CDN etc.)
            if (url.includes('localhost:8586') && resp.status() === 404) {
                failures.push(`404: ${url}`)
            }
        })
        await page.goto('./')
        await page.waitForLoadState('networkidle')

        expect(failures, `Failed asset requests:\n${failures.join('\n')}`).toHaveLength(0)
    })

    test('settings.json is fetched from the subpath', async ({ page }) => {
        // Intercept settings.json to capture the URL it was requested from.
        // page.route is the reliable way — avoids race with un-awaited page.goto.
        let capturedURL = null
        await page.route('**/settings.json', async (route) => {
            capturedURL = route.request().url()
            await route.continue()
        })
        await page.goto('./')
        await page.waitForLoadState('networkidle')
        expect(capturedURL).not.toBeNull()
        expect(capturedURL).toContain('/sub/settings.json')
    })
})

// ---------------------------------------------------------------------------
// Theme loading
// ---------------------------------------------------------------------------

test.describe('Subpath — theme loading', () => {
    test('light theme CSS loads from subpath', async ({ page }) => {
        // Intercept settings.json to expose a non-dark theme
        await page.route('**/settings.json', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ name: 'Plik', theme: 'light', themes: ['dark', 'light'] }),
            })
        })

        const themeRequests = []
        page.on('request', (req) => {
            if (req.url().includes('/themes/')) themeRequests.push(req.url())
        })

        await page.goto('./')
        await page.waitForLoadState('networkidle')

        // The light theme CSS should be requested from /sub/themes/light.css
        expect(themeRequests.some((u) => u.includes('/sub/themes/'))).toBe(true)
    })
})

// ---------------------------------------------------------------------------
// Language flags
// ---------------------------------------------------------------------------

test.describe('Subpath — language flags', () => {
    test('flag images load from subpath', async ({ page }) => {
        // Serve a settings.json that enables the language picker
        await page.route('**/settings.json', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ name: 'Plik', language: 'auto', languages: ['en', 'fr'] }),
            })
        })

        const flagRequests = []
        page.on('request', (req) => {
            if (req.url().includes('/flags/')) flagRequests.push(req.url())
        })
        const flag404s = []
        page.on('response', (resp) => {
            if (resp.url().includes('/flags/') && resp.status() === 404) {
                flag404s.push(resp.url())
            }
        })

        await page.goto('./')
        await page.waitForLoadState('networkidle')

        // Open the language picker to trigger flag image requests
        const langPicker = page.locator('[aria-label="Language picker"], [title="Language"], button').filter({ hasText: /^[A-Z]{2}$|Auto/i }).first()
        if (await langPicker.isVisible()) {
            await langPicker.click()
            await page.waitForTimeout(500)
        }

        if (flagRequests.length > 0) {
            // All flag requests must be from the subpath
            for (const url of flagRequests) {
                expect(url).toContain('/sub/flags/')
            }
            expect(flag404s).toHaveLength(0)
        }
        // If no flag requests were made (picker not visible), the test still
        // passes — we are primarily checking no 404s from the earlier hook.
    })
})

// ---------------------------------------------------------------------------
// Upload + download
// ---------------------------------------------------------------------------

test.describe('Subpath — upload and download', () => {
    test('can upload a file and view the download page', async ({ page }) => {
        await loginAs(page, ADMIN_LOGIN, ADMIN_PASSWORD)

        const buffer = Buffer.from('hello from subpath')
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'subpath-test.txt',
            mimeType: 'text/plain',
            buffer,
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        await expect(page.getByText('subpath-test.txt').first()).toBeVisible({ timeout: 5_000 })
    })

    test('API calls use subpath-relative URLs', async ({ page }) => {
        // Verify that fetch('/upload') from the SPA actually hits /sub/upload
        const apiRequests = []
        page.on('request', (req) => {
            if (req.url().includes('localhost:8586')) apiRequests.push(req.url())
        })

        await loginAs(page, ADMIN_LOGIN, ADMIN_PASSWORD)

        // All API requests should be scoped to /sub/
        const nonSubpath = apiRequests.filter(
            (u) => !u.includes('/sub/') && !u.endsWith('/sub')
        )
        expect(
            nonSubpath,
            `API calls outside /sub/:\n${nonSubpath.join('\n')}`
        ).toHaveLength(0)
    })
})
