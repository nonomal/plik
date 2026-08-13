import { test, expect, uploadTestFile } from './fixtures.js'

test.describe('Upload flow', () => {
    test('shows upload drop zone on root page', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // The drop zone prompt should be visible
        await expect(page.getByText('Drop, paste or click to select files')).toBeVisible()
    })

    test('uploads a single file via file input', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Set a file on the hidden input
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'hello.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('hello world'),
        })

        // File should appear in the pending list
        await expect(page.getByText('hello.txt')).toBeVisible()

        // Click the green Upload button (exact match avoids "Create empty upload")
        await page.getByRole('button', { name: 'Upload', exact: true }).click()

        // Should navigate to download view
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Download view should show the filename
        await expect(page.getByText('hello.txt').first()).toBeVisible()
    })

    test('uploads multiple files', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const input = page.locator('input[type="file"]')
        await input.setInputFiles([
            { name: 'a.txt', mimeType: 'text/plain', buffer: Buffer.from('aaa') },
            { name: 'b.txt', mimeType: 'text/plain', buffer: Buffer.from('bbb') },
            { name: 'c.txt', mimeType: 'text/plain', buffer: Buffer.from('ccc') },
        ])

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // All three files should appear
        await expect(page.getByText('a.txt')).toBeVisible()
        await expect(page.getByText('b.txt')).toBeVisible()
        await expect(page.getByText('c.txt')).toBeVisible()
    })

    test('text paste mode creates a file', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Click the "Paste text" link
        await page.getByText('Paste text').click()

        // Text mode should open — find the editor area
        const editor = page.locator('.cm-content, textarea').first()
        await editor.waitFor({ state: 'visible', timeout: 5_000 })
        await editor.fill('console.log("hello")')

        // Add as file
        await page.getByRole('button', { name: /^Add$/i }).click()

        // File should appear in the pending list
        await expect(page.getByText(/paste\.\w+/)).toBeVisible()

        // Upload it
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Should be on download page
        await expect(page.getByRole('link', { name: /paste\.\w+/ })).toBeVisible()
    })

    test('clipboard paste opens text editor with pasted content', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const pastedText = 'Hello from clipboard paste!'

        // Simulate a paste event with text data on the main container
        await page.evaluate((text) => {
            const event = new ClipboardEvent('paste', {
                bubbles: true,
                cancelable: true,
                clipboardData: new DataTransfer(),
            })
            event.clipboardData.setData('text/plain', text)
            document.querySelector('.flex.flex-col.md\\:flex-row').dispatchEvent(event)
        }, pastedText)

        // Text editor should open with the pasted content
        const editor = page.locator('.cm-content, textarea').first()
        await expect(editor).toBeVisible({ timeout: 5_000 })
        await expect(editor).toContainText(pastedText)

        // Add as file and upload
        await page.getByRole('button', { name: /^Add$/i }).click()
        await expect(page.getByText(/paste\.txt/)).toBeVisible()

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Verify the file is on the download page with correct content
        await expect(page.getByRole('link', { name: /paste\.txt/ })).toBeVisible()

        // The text viewer should auto-open (single text file) with the pasted content
        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })
        await expect(panel).toContainText(pastedText)
    })

    test('uploads a file with special characters in name', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Filename with #, parentheses, and spaces — previously caused 404
        const fileName = '#test (special-chars).txt'
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: fileName,
            mimeType: 'text/plain',
            buffer: Buffer.from('special content'),
        })

        await expect(page.getByText(fileName)).toBeVisible()

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Verify the file appears on the download page with the original name
        await expect(page.getByText(fileName).first()).toBeVisible()
    })

    test('user-edited filename is preserved despite auto-detection', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Open text paste mode
        await page.getByText('Paste text').click()

        const editor = page.locator('.cm-content, textarea').first()
        await editor.waitFor({ state: 'visible', timeout: 5_000 })

        // Type content that looks like JavaScript (triggers language detection)
        await editor.fill('function hello() {\n  console.log("world")\n}\nhello()')

        // Wait for the auto-detection to fire and update the filename
        const filenameInput = page.locator('input[placeholder="paste.txt"]')
        await expect(filenameInput).toHaveValue(/paste\.\w+/, { timeout: 5_000 })

        // User manually edits the filename
        await filenameInput.fill('my-notes.txt')
        await expect(filenameInput).toHaveValue('my-notes.txt')

        // Wait to make sure auto-detection doesn't overwrite it
        await page.waitForTimeout(2000)
        await expect(filenameInput).toHaveValue('my-notes.txt')

        // Add as file and upload
        await page.getByRole('button', { name: /^Add$/i }).click()
        await expect(page.getByText('my-notes.txt')).toBeVisible()

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Download page should show the user-chosen filename
        await expect(page.getByRole('link', { name: 'my-notes.txt' })).toBeVisible()
    })
})

test.describe('Paste editor markdown preview', () => {
    test('shows Code/Preview tabs when filename ends with .md', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Open text paste mode
        await page.getByText('Paste text').click()

        // Type markdown content
        const editor = page.locator('.cm-content, textarea').first()
        await editor.waitFor({ state: 'visible', timeout: 5_000 })
        await editor.fill('# Hello\n\nThis is **bold** text.')

        // No tabs yet (default filename is paste.txt)
        await expect(page.getByRole('button', { name: 'Preview' })).not.toBeVisible()

        // Rename to .md — tabs should appear
        const filenameInput = page.locator('input[placeholder="paste.txt"]')
        await filenameInput.fill('notes.md')

        const previewBtn = page.getByRole('button', { name: 'Preview' })
        const codeBtn = page.getByRole('button', { name: 'Code' })
        await expect(previewBtn).toBeVisible()
        await expect(codeBtn).toBeVisible()

        // Switch to Preview — rendered markdown should be visible
        await previewBtn.click()
        await expect(page.locator('.prose strong')).toHaveText('bold')

        // Switch back to Code — editor is visible again
        await codeBtn.click()
        await expect(editor).toBeVisible()
    })

    test('does NOT show tabs for .txt filename', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.getByText('Paste text').click()

        const editor = page.locator('.cm-content, textarea').first()
        await editor.waitFor({ state: 'visible', timeout: 5_000 })
        await editor.fill('plain text content')

        // Default filename is paste.txt — no tabs
        await expect(page.getByRole('button', { name: 'Preview' })).not.toBeVisible()
        await expect(page.getByRole('button', { name: 'Code' })).not.toBeVisible()
    })
})
