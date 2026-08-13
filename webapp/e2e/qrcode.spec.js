import { test, expect, uploadTestFile } from './fixtures.js'

test.describe('QR Code', () => {
    test('upload QR code dialog opens', async ({ page }) => {
        await uploadTestFile(page)

        // Click QR Code button in sidebar
        await page.getByRole('button', { name: /QR Code/i }).click()

        // Dialog should open with "Upload Link" title and a QR code image
        await expect(page.getByText('Upload Link')).toBeVisible({ timeout: 3_000 })
        const qrImg = page.locator('img[src*="/qrcode"]')
        await expect(qrImg).toBeVisible()
    })

    test('upload QR code image loads', async ({ page }) => {
        await uploadTestFile(page)
        await page.getByRole('button', { name: /QR Code/i }).click()

        // Verify the QR code image actually loads (naturalWidth > 0)
        const qrImg = page.locator('img[src*="/qrcode"]')
        await expect(qrImg).toBeVisible({ timeout: 5_000 })
        const naturalWidth = await qrImg.evaluate(img => img.naturalWidth)
        expect(naturalWidth).toBeGreaterThan(0)
    })

    test('QR dialog closes on X button', async ({ page }) => {
        await uploadTestFile(page)
        await page.getByRole('button', { name: /QR Code/i }).click()
        await expect(page.getByText('Upload Link')).toBeVisible({ timeout: 3_000 })

        // Click the close button (× button)
        const closeBtn = page.locator('.fixed.inset-0.z-50').getByRole('button').first()
        await closeBtn.click()

        // Dialog should be hidden
        await expect(page.locator('img[src*="/qrcode"]')).not.toBeVisible()
    })

    test('per-file QR code opens', async ({ page }) => {
        await uploadTestFile(page, 'qr-test-file.txt', 'qr test content')

        // Click QR button on the file row (title="Show QR code")
        const qrBtn = page.getByTitle('Show QR code').first()
        await expect(qrBtn).toBeVisible({ timeout: 3_000 })
        await qrBtn.click()

        // Dialog should show with QR code image
        await expect(page.locator('img[src*="/qrcode"]')).toBeVisible({ timeout: 3_000 })
    })

    test('QR dialog closes on backdrop click', async ({ page }) => {
        await uploadTestFile(page)
        await page.getByRole('button', { name: /QR Code/i }).click()
        await expect(page.locator('img[src*="/qrcode"]')).toBeVisible({ timeout: 3_000 })

        // Click the backdrop overlay
        const overlay = page.locator('.fixed.inset-0.z-50')
        await overlay.click({ position: { x: 5, y: 5 }, force: true })

        // Dialog should close
        await expect(page.locator('img[src*="/qrcode"]')).not.toBeVisible({ timeout: 3_000 })
    })
})
