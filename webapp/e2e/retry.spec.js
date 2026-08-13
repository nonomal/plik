import { test, expect, uploadTestFile } from './fixtures.js'

/**
 * Tests for upload failure handling and retry functionality.
 *
 * Strategy: intercept POST /file/* requests via page.route() to simulate
 * server errors, verify the error UI, then unroute and retry.
 */
test.describe('Upload failure and retry', () => {
    test('failed upload shows error message and Retry button', async ({ page }) => {
        // Intercept file upload XHR with a 500 error
        await page.route('**/file/**', (route) => {
            if (route.request().method() === 'POST') {
                return route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    body: JSON.stringify({ message: 'Internal Server Error' }),
                })
            }
            return route.continue()
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'fail-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('will fail'),
        })

        // Click Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()

        // Wait for navigation to download view
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // File should show error message and Retry button
        await expect(page.getByText(/upload failed|internal server error/i).first()).toBeVisible({ timeout: 10_000 })
        await expect(page.getByRole('button', { name: 'Retry' }).first()).toBeVisible()
    })

    test('Retry button re-uploads the file successfully', async ({ page }) => {
        // Intercept to fail on first attempt
        let shouldFail = true
        await page.route('**/file/**', (route) => {
            if (route.request().method() === 'POST' && shouldFail) {
                return route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    body: JSON.stringify({ message: 'Temporary failure' }),
                })
            }
            return route.continue()
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'retry-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('retry me'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Wait for error
        await expect(page.getByRole('button', { name: 'Retry' }).first()).toBeVisible({ timeout: 10_000 })

        // Remove the failure intercept and retry
        shouldFail = false
        await page.getByRole('button', { name: 'Retry' }).first().click()

        // File should upload and error should disappear
        await expect(page.getByRole('button', { name: 'Retry' })).not.toBeVisible({ timeout: 10_000 })

        // The file should now be in the uploaded files list
        await expect(page.getByText('retry-test.txt').first()).toBeVisible()
    })

    test('Retry Failed button retries all failed files', async ({ page }) => {
        // Intercept to fail on all initial attempts
        let shouldFail = true
        await page.route('**/file/**', (route) => {
            if (route.request().method() === 'POST' && shouldFail) {
                return route.fulfill({
                    status: 500,
                    contentType: 'application/json',
                    body: JSON.stringify({ message: 'Batch failure' }),
                })
            }
            return route.continue()
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload 2 files
        const input = page.locator('input[type="file"]')
        await input.setInputFiles([
            { name: 'batch1.txt', mimeType: 'text/plain', buffer: Buffer.from('file 1') },
            { name: 'batch2.txt', mimeType: 'text/plain', buffer: Buffer.from('file 2') },
        ])

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Wait for both to fail
        await expect(page.getByRole('button', { name: 'Retry' }).first()).toBeVisible({ timeout: 10_000 })

        // "Retry Failed" button should appear
        const retryAllBtn = page.getByRole('button', { name: /Retry Failed/i })
        await expect(retryAllBtn).toBeVisible({ timeout: 5_000 })

        // Remove failure and click Retry Failed
        shouldFail = false
        await retryAllBtn.click()

        // All retry buttons should disappear
        await expect(page.getByRole('button', { name: 'Retry' }).first()).not.toBeVisible({ timeout: 10_000 })

        // Both files should be uploaded
        await expect(page.getByText('batch1.txt').first()).toBeVisible()
        await expect(page.getByText('batch2.txt').first()).toBeVisible()
    })

    test('Cancel button stops uploading', async ({ page }) => {
        // Use a slow route to keep the upload in-progress long enough to cancel
        await page.route('**/file/**', async (route) => {
            if (route.request().method() === 'POST') {
                // Delay response to keep upload "in progress"
                await new Promise(resolve => setTimeout(resolve, 30_000))
                return route.fulfill({ status: 200, body: '{}' })
            }
            return route.continue()
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'cancel-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('cancel me'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })

        // Look for Cancel button on the file row during upload
        const cancelBtn = page.getByRole('button', { name: /Cancel/i }).first()
        await expect(cancelBtn).toBeVisible({ timeout: 10_000 })

        // Click cancel
        await cancelBtn.click()

        // The cancel button should disappear
        await expect(cancelBtn).not.toBeVisible({ timeout: 5_000 })
    })
})
