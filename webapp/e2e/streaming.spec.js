import { test, expect } from './fixtures.js'

/**
 * Tests for streaming upload functionality.
 *
 * Streaming uploads use /stream/ URL path instead of /file/,
 * and have different UI behavior (no zip, no add files, waiting badge).
 */
test.describe('Streaming uploads', () => {
    test('stream toggle creates streaming upload', async ({ page, withConfig }) => {
        await withConfig({ feature_stream: 'enabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable the Streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const toggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await expect(toggle).toBeVisible({ timeout: 5_000 })
        await toggle.click()

        // Add a file — but intercept the upload XHR to verify stream path and avoid hanging
        let uploadUrl = ''
        await page.route('**/stream/**', async (route) => {
            if (route.request().method() === 'POST') {
                uploadUrl = route.request().url()
                // Fulfill immediately to avoid the stream blocking
                return route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({ id: 'fake-file-id', fileName: 'stream-test.txt' }),
                })
            }
            return route.continue()
        })

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'stream-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('streaming content'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })

        // Wait for the stream upload request to fire
        await page.waitForTimeout(3_000)

        // Verify the upload used the /stream/ URL path
        expect(uploadUrl).toContain('/stream/')
    })

    test('Zip Archive hidden for streaming uploads', async ({ page, withConfig }) => {
        await withConfig({ feature_stream: 'enabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const toggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await toggle.click()

        // Intercept stream upload to avoid hanging
        await page.route('**/stream/**', (route) => {
            if (route.request().method() === 'POST') {
                return route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({ id: 'fake', fileName: 'zip-test.txt' }),
                })
            }
            return route.continue()
        })

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'zip-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('zipping?'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Zip Archive button should NOT be visible for streaming uploads
        await expect(page.getByText(/Zip Archive/i)).not.toBeVisible()
    })

    test('Add Files hidden for streaming uploads', async ({ page, withConfig }) => {
        await withConfig({ feature_stream: 'enabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const toggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await toggle.click()

        // Intercept stream upload
        await page.route('**/stream/**', (route) => {
            if (route.request().method() === 'POST') {
                return route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({ id: 'fake', fileName: 'add-test.txt' }),
                })
            }
            return route.continue()
        })

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'add-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('add more?'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Add Files button should NOT be visible for streaming uploads
        await expect(page.getByRole('button', { name: /Add Files/i })).not.toBeVisible()
    })

    test('stream upload uses /stream/ URL path', async ({ page, withConfig }) => {
        await withConfig({ feature_stream: 'enabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const toggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await toggle.click()

        // Track the upload URL
        let capturedUrl = ''
        await page.route('**/stream/**', async (route) => {
            if (route.request().method() === 'POST') {
                capturedUrl = route.request().url()
                return route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({ id: 'captured', fileName: 'path-test.txt' }),
                })
            }
            return route.continue()
        })

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'path-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('path check'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForTimeout(3_000)

        // URL should use /stream/ path, not /file/
        expect(capturedUrl).toContain('/stream/')
        expect(capturedUrl).not.toContain('/file/')
    })

    test('View button hidden for streaming uploads', async ({ page, withConfig }) => {
        await withConfig({ feature_stream: 'enabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const toggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await toggle.click()

        // Intercept stream upload to avoid hanging (streaming blocks until consumed)
        await page.route('**/stream/**', (route) => {
            if (route.request().method() === 'POST') {
                return route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({ id: 'fake', fileName: 'stream-view.txt' }),
                })
            }
            return route.continue()
        })

        // Upload a text file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'stream-view.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('streaming text content'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // View button should NOT be visible for streaming uploads
        await expect(page.getByRole('button', { name: 'View' })).not.toBeVisible()
    })

    test('text viewer does NOT auto-open for streaming uploads', async ({ page, withConfig }) => {
        await withConfig({ feature_stream: 'enabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const toggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await toggle.click()

        // Intercept stream upload to avoid hanging (streaming blocks until consumed)
        await page.route('**/stream/**', (route) => {
            if (route.request().method() === 'POST') {
                return route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({ id: 'fake', fileName: 'stream-auto.txt' }),
                })
            }
            return route.continue()
        })

        // Upload a single text file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'stream-auto.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('should not auto open'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Viewer panel should NOT auto-open for streaming uploads
        await expect(page.locator('#file-viewer-panel')).not.toBeVisible()
    })
})
