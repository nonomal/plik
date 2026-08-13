import { test, expect } from './fixtures.js'

/**
 * Tests for Stream + E2EE combined functionality.
 *
 * When both streaming and E2EE are enabled:
 * - Upload POST goes to /stream/ (blocks until a downloader arrives)
 * - Decrypt button triggers fetch() GET to /stream/ (unblocks the upload pipe)
 * - Data flows: uploader → server pipe → downloader → age decrypt → blob download
 *
 * This combination previously caused ERR_CONTENT_LENGTH_MISMATCH because:
 * 1. The server set Content-Length from the original (unencrypted) file size
 * 2. But the actual data piped was the encrypted blob (larger due to age overhead)
 * 3. The download URL was also using /file/ instead of /stream/
 */
test.describe('Stream + E2EE', () => {
    test.beforeEach(async ({ withConfig }) => {
        await withConfig({ feature_stream: 'enabled', feature_e2ee: 'enabled' })
    })

    test('stream + E2EE upload uses /stream/ endpoint and data is encrypted', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable Streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const streamToggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await streamToggle.click()

        // Enable E2EE toggle
        const e2eeSection = page.getByText('End-to-End Encryption').locator('xpath=ancestor::div[1]')
        const e2eeToggle = e2eeSection.locator('.toggle-switch')
        await e2eeToggle.click()

        // Capture the generated passphrase
        const passphraseInput = page.locator('aside input.font-mono').first()
        await expect(passphraseInput).toBeVisible({ timeout: 3_000 })

        // Track the upload URL and body to verify it uses /stream/ and data is encrypted
        let uploadUrl = ''
        let uploadBodyText = ''
        await page.route('**/stream/**', async (route) => {
            if (route.request().method() === 'POST') {
                uploadUrl = route.request().url()
                const body = route.request().postDataBuffer()
                if (body) {
                    uploadBodyText = body.toString('utf-8')
                }
                return route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({ id: 'fake-stream-e2ee', fileName: 'secret.txt' }),
                })
            }
            return route.continue()
        })

        // Upload a file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'secret.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('this should be encrypted not plaintext'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 15_000 })
        await page.waitForTimeout(3_000)

        // Upload should use /stream/ endpoint, not /file/
        expect(uploadUrl).toContain('/stream/')
        expect(uploadUrl).not.toContain('/file/')

        // Upload body should contain the age encryption header (encrypted, not plaintext)
        if (uploadBodyText) {
            expect(uploadBodyText).toContain('age-encryption.org')
            expect(uploadBodyText).not.toContain('this should be encrypted not plaintext')
        }
    })

    test('stream + E2EE shows both badges', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable Streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const streamToggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await streamToggle.click()

        // Enable E2EE toggle
        const e2eeSection = page.getByText('End-to-End Encryption').locator('xpath=ancestor::div[1]')
        const e2eeToggle = e2eeSection.locator('.toggle-switch')
        await e2eeToggle.click()

        await expect(page.locator('aside input.font-mono').first()).toBeVisible({ timeout: 3_000 })

        // Intercept stream upload to avoid blocking
        await page.route('**/stream/**', async (route) => {
            if (route.request().method() === 'POST') {
                return route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({ id: 'fake', fileName: 'badges.txt' }),
                })
            }
            return route.continue()
        })

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'badges.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('badge test'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 15_000 })
        await page.waitForLoadState('networkidle')

        // Both stream and E2EE badges should be visible
        await expect(page.getByText('Stream').first()).toBeVisible({ timeout: 5_000 })
        await expect(page.locator('text=End-to-End Encrypted').first()).toBeVisible({ timeout: 5_000 })
    })

    test('stream + E2EE full roundtrip: upload encrypted, decrypt download matches original', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable Streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const streamToggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await streamToggle.click()

        // Enable E2EE toggle
        const e2eeSection = page.getByText('End-to-End Encryption').locator('xpath=ancestor::div[1]')
        const e2eeToggle = e2eeSection.locator('.toggle-switch')
        await e2eeToggle.click()

        // Wait for passphrase
        const passphraseInput = page.locator('aside input.font-mono').first()
        await expect(passphraseInput).toBeVisible({ timeout: 3_000 })
        const passphrase = await passphraseInput.inputValue()
        expect(passphrase.length).toBeGreaterThan(0)

        // Verify the upload request body is encrypted (contains age header)
        let uploadBodyText = ''
        page.on('request', async (req) => {
            if (req.url().includes('/stream/') && req.method() === 'POST') {
                const body = req.postDataBuffer()
                if (body) uploadBodyText = body.toString('utf-8')
            }
        })

        // Upload a file (real — no interception, full server roundtrip)
        const originalContent = 'This is secret stream+e2ee content for roundtrip test'
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'stream-secret.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from(originalContent),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 15_000 })
        await page.waitForLoadState('networkidle')

        // Verify uploaded data was encrypted (age header present, plaintext absent)
        if (uploadBodyText) {
            expect(uploadBodyText).toContain('age-encryption.org')
            expect(uploadBodyText).not.toContain(originalContent)
        }

        // Decrypt button should be visible (not regular download)
        const decryptBtn = page.getByRole('button', { name: /Decrypt/i }).first()
        await expect(decryptBtn).toBeVisible({ timeout: 10_000 })

        // Click decrypt — triggers fetchAndDecrypt → GET /stream/ → unblocks upload → decrypts
        const downloadPromise = page.waitForEvent('download', { timeout: 30_000 })
        await decryptBtn.click()

        // Wait for the download to complete
        const download = await downloadPromise

        // Verify the downloaded file has the correct name
        expect(download.suggestedFilename()).toBe('stream-secret.txt')

        // Read the downloaded content and verify it matches the original (decryption succeeded)
        const downloadPath = await download.path()
        expect(downloadPath).toBeTruthy()
        const fs = await import('fs')
        const downloadedContent = fs.readFileSync(downloadPath, 'utf-8')
        expect(downloadedContent).toBe(originalContent)

        // No error banner should be visible
        await expect(page.locator('text=Decryption failed')).not.toBeVisible()
    })

    test('stream + E2EE download fetch uses /stream/ path', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable Streaming toggle
        const streamLabel = page.getByText('Streaming', { exact: false }).first()
        const streamToggle = streamLabel.locator('xpath=..').locator('.toggle-switch')
        await streamToggle.click()

        // Enable E2EE toggle
        const e2eeSection = page.getByText('End-to-End Encryption').locator('xpath=ancestor::div[1]')
        const e2eeToggle = e2eeSection.locator('.toggle-switch')
        await e2eeToggle.click()

        await expect(page.locator('aside input.font-mono').first()).toBeVisible({ timeout: 3_000 })

        // Track the download fetch URL
        let downloadUrl = ''
        page.on('request', (req) => {
            if (req.url().includes('/stream/') && req.method() === 'GET' &&
                !req.url().includes('/upload/')) {
                downloadUrl = req.url()
            }
        })

        // Upload (real, through server)
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'url-check.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('URL path verification content'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 15_000 })
        await page.waitForLoadState('networkidle')

        // Click decrypt to trigger the GET /stream/ fetch
        const decryptBtn = page.getByRole('button', { name: /Decrypt/i }).first()
        await expect(decryptBtn).toBeVisible({ timeout: 10_000 })

        const downloadPromise = page.waitForEvent('download', { timeout: 30_000 })
        await decryptBtn.click()
        await downloadPromise

        // Verify the download used /stream/ not /file/
        expect(downloadUrl).toContain('/stream/')
        expect(downloadUrl).not.toContain('/file/')
    })
})
