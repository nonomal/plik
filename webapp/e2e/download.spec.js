import { test, expect, uploadTestFile } from './fixtures.js'

test.describe('Download view', () => {
    test('shows file list after upload', async ({ page }) => {
        await uploadTestFile(page, 'readme.txt', 'file content here')

        // Should show the file in the download view
        await expect(page.getByRole('link', { name: 'readme.txt' })).toBeVisible()

        // Should show file count as an h3 heading (e.g. "1 file")
        await expect(page.getByRole('heading', { name: /\d+ files?/ })).toBeVisible({ timeout: 5_000 })

        // No comment was set — comment section should not be visible
        await expect(page.getByRole('heading', { name: 'Comment' })).not.toBeVisible()
    })

    test('file download link is present', async ({ page }) => {
        await uploadTestFile(page, 'download-me.txt', 'download content')

        // The file row should contain a download link/button
        // FileRow component renders the file name as a clickable link
        const fileLink = page.getByRole('link', { name: 'download-me.txt' })
            .or(page.getByText('download-me.txt'))
        await expect(fileLink.first()).toBeVisible()
    })

    test('sidebar shows upload metadata', async ({ page }) => {
        await uploadTestFile(page, 'meta-test.txt', 'metadata test content')

        // Download sidebar should show expiration info
        await expect(page.getByText(/expire|remaining|never/i).first()).toBeVisible()
    })

    test('share URL is copyable', async ({ page }) => {
        await uploadTestFile(page, 'share-test.txt', 'share content')

        // The sidebar should have a "Share" section or copy link button
        const shareSection = page.getByText(/share|link/i)
        await expect(shareSection.first()).toBeVisible()
    })

    test('upload comment is displayed on the download page', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'commented.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('file with comment'),
        })

        // Enable comment toggle and write a comment
        const toggle = page.getByText('Comment').first().locator('xpath=..').locator('.toggle-switch')
        await toggle.click()
        await page.locator('textarea').fill('This is a **test comment**')

        // Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // The download view should show the comment section
        await expect(page.getByRole('heading', { name: 'Comment' })).toBeVisible({ timeout: 5_000 })
        // The rendered markdown should contain the bold text
        await expect(page.locator('.prose strong')).toHaveText('test comment')
    })
})

test.describe('Text viewer', () => {
    test('shows file content when View button is clicked', async ({ page }) => {
        const fileContent = 'Hello from the text viewer test'
        await uploadTestFile(page, 'viewer-test.txt', fileContent)

        const panel = page.locator('#file-viewer-panel')
        const viewBtn = page.getByRole('button', { name: 'View', exact: true })

        // Single text file auto-opens the viewer — wait for it to fully render
        await expect(panel).toBeVisible({ timeout: 5_000 })

        // Click View to toggle it off
        await viewBtn.click()
        await expect(panel).not.toBeVisible()

        // Click View again — this time it's a manual open
        await viewBtn.click()

        // Viewer panel should re-appear with the correct content
        await expect(panel).toBeVisible({ timeout: 5_000 })
        await expect(panel).toContainText(fileContent)
    })

    test('auto-opens for a single text file upload', async ({ page }) => {
        const fileContent = 'Auto-viewed single file content'
        await uploadTestFile(page, 'auto-view.txt', fileContent)

        // Viewer panel should appear automatically
        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })
        await expect(panel).toContainText(fileContent)
        // Panel header should show the filename
        await expect(panel).toContainText('auto-view.txt')
    })

    test('does NOT auto-open for multiple file uploads', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add two text files
        const input = page.locator('input[type="file"]')
        await input.setInputFiles([
            { name: 'file1.txt', mimeType: 'text/plain', buffer: Buffer.from('content one') },
            { name: 'file2.txt', mimeType: 'text/plain', buffer: Buffer.from('content two') },
        ])

        // Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Both files should be listed
        await expect(page.getByRole('link', { name: 'file1.txt' })).toBeVisible()
        await expect(page.getByRole('link', { name: 'file2.txt' })).toBeVisible()

        // Viewer panel should NOT be open automatically
        await expect(page.locator('#file-viewer-panel')).not.toBeVisible()
    })

    test('View button is not shown for non-viewable uploads', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload a binary file that is neither text nor image
        const input = page.locator('input[type="file"]')
        // Minimal ZIP file (local file header + EOCD)
        const zip = Buffer.from(
            '504b05060000000000000000000000000000000000000000',
            'hex'
        )
        await input.setInputFiles({
            name: 'archive.zip',
            mimeType: 'application/zip',
            buffer: zip,
        })

        // Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // File should be listed
        await expect(page.getByRole('link', { name: 'archive.zip' })).toBeVisible()

        // No "View" button should be present (only appears for text/image files)
        await expect(page.getByRole('button', { name: 'View' })).not.toBeVisible()

        // Viewer panel should not be open
        await expect(page.locator('#file-viewer-panel')).not.toBeVisible()
    })

    test('View button hidden for one-shot uploads', async ({ page, withConfig }) => {
        await withConfig({ feature_one_shot: 'default' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload a text file (one-shot is on by default via config)
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'oneshot.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('one shot content'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // File should be listed
        await expect(page.getByRole('link', { name: 'oneshot.txt' })).toBeVisible()

        // View button should NOT be visible for one-shot uploads
        await expect(page.getByRole('button', { name: 'View' })).not.toBeVisible()
    })

    test('does NOT auto-open for one-shot uploads', async ({ page, withConfig }) => {
        await withConfig({ feature_one_shot: 'default' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload a single text file with one-shot enabled
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'oneshot-auto.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('should not auto open'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // File should be listed
        await expect(page.getByRole('link', { name: 'oneshot-auto.txt' })).toBeVisible()

        // Viewer panel should NOT auto-open for one-shot uploads
        await expect(page.locator('#file-viewer-panel')).not.toBeVisible()
    })
})

test.describe('Markdown preview', () => {
    test('shows Code/Preview tabs for .md files and defaults to Preview', async ({ page }) => {
        const mdContent = '# Hello World\n\nThis is **bold text** and *italic*.'
        await uploadTestFile(page, 'readme.md', mdContent)

        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })

        // Preview tab should be active by default for markdown files
        const previewBtn = panel.getByRole('button', { name: 'Preview' })
        const codeBtn = panel.getByRole('button', { name: 'Code' })
        await expect(previewBtn).toBeVisible({ timeout: 5_000 })
        await expect(codeBtn).toBeVisible()

        // Rendered markdown should be visible (default tab is preview)
        await expect(panel.locator('.prose strong')).toHaveText('bold text')

        // Switch to Code tab — CodeMirror source should be visible
        await codeBtn.click()
        await expect(panel).toContainText('# Hello World')

        // Switch back to Preview
        await previewBtn.click()
        await expect(panel.locator('.prose strong')).toHaveText('bold text')
    })

    test('does NOT show tabs for .txt files', async ({ page }) => {
        await uploadTestFile(page, 'plain.txt', 'just plain text')

        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })

        // Code/Preview tabs should NOT be present
        await expect(panel.getByRole('button', { name: 'Preview' })).not.toBeVisible()
        await expect(panel.getByRole('button', { name: 'Code' })).not.toBeVisible()

        // Content should still be shown in the editor
        await expect(panel).toContainText('just plain text')
    })
})

test.describe('Image viewer', () => {
    // Minimal valid 1x1 red PNG (67 bytes)
    const PNG_BYTES = Buffer.from(
        '89504e470d0a1a0a0000000d4948445200000001000000010802000000907753' +
        'de0000000c4944415408d763f86f0000000200018dcc2fe60000000049454e44ae426082',
        'hex'
    )

    async function uploadImageFile(page, filename = 'test.png') {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: filename,
            mimeType: 'image/png',
            buffer: PNG_BYTES,
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')
    }

    test('auto-opens image viewer with img tag for single image upload', async ({ page }) => {
        await uploadImageFile(page, 'photo.png')

        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })

        // Image should be rendered as an <img> tag
        const img = panel.locator('img')
        await expect(img).toBeVisible()
        await expect(img).toHaveAttribute('alt', 'photo.png')

        // Should NOT have CodeEditor or markdown tabs
        await expect(panel.locator('.cm-editor')).not.toBeVisible()
        await expect(panel.getByRole('button', { name: 'Preview' })).not.toBeVisible()
    })

    test('View button appears for image files', async ({ page }) => {
        await uploadImageFile(page, 'icon.png')

        // The View button should be visible on the file row
        const viewBtn = page.getByRole('button', { name: 'View', exact: true })
        await expect(viewBtn).toBeVisible({ timeout: 5_000 })
    })
})

test.describe('Viewer navigation', () => {
    test('navigates between viewable files with arrows and keyboard', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload 3 files: 2 text + 1 image
        const input = page.locator('input[type="file"]')
        await input.setInputFiles([
            { name: 'a.txt', mimeType: 'text/plain', buffer: Buffer.from('file A') },
            { name: 'b.txt', mimeType: 'text/plain', buffer: Buffer.from('file B') },
            {
                name: 'c.png', mimeType: 'image/png', buffer: Buffer.from(
                    '89504e470d0a1a0a0000000d4948445200000001000000010802000000907753' +
                    'de0000000c4944415408d763f86f0000000200018dcc2fe60000000049454e44ae426082',
                    'hex'
                )
            },
        ])

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Viewer should NOT auto-open (multiple files)
        const panel = page.locator('#file-viewer-panel')
        await expect(panel).not.toBeVisible()

        // Click View on first file
        const viewButtons = page.getByRole('button', { name: 'View', exact: true })
        await expect(viewButtons.first()).toBeVisible({ timeout: 5_000 })
        await viewButtons.first().click()

        await expect(panel).toBeVisible({ timeout: 5_000 })

        // Position indicator should show 1/3
        await expect(panel.locator('text=1/3')).toBeVisible()

        // Prev should be disabled, Next enabled
        const prevBtn = panel.getByTitle('Previous file (←)')
        const nextBtn = panel.getByTitle('Next file (→)')
        await expect(prevBtn).toBeDisabled()
        await expect(nextBtn).not.toBeDisabled()

        // Click Next → should show file 2/3
        await nextBtn.click()
        await expect(panel.locator('text=2/3')).toBeVisible()
        await expect(prevBtn).not.toBeDisabled()

        // Keyboard ArrowRight → 3/3
        await page.keyboard.press('ArrowRight')
        await expect(panel.locator('text=3/3')).toBeVisible()
        await expect(nextBtn).toBeDisabled()

        // Keyboard ArrowLeft → 2/3
        await page.keyboard.press('ArrowLeft')
        await expect(panel.locator('text=2/3')).toBeVisible()

        // Escape closes viewer
        await page.keyboard.press('Escape')
        await expect(panel).not.toBeVisible()
    })
})

test.describe('Add files', () => {
    test('Add Files button visible for admin', async ({ page }) => {
        await uploadTestFile(page)

        // Admin (has uploadToken) should see the "Add Files" button
        await expect(page.getByRole('button', { name: /Add Files/i })).toBeVisible({ timeout: 5_000 })
    })

    test('add files stages and uploads', async ({ page }) => {
        await uploadTestFile(page, 'first.txt', 'first file')

        // Click "Add Files" — triggers the hidden file input
        const addBtn = page.getByRole('button', { name: /Add Files/i })
        await expect(addBtn).toBeVisible({ timeout: 5_000 })

        // Add a second file via the input
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'second.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('second file content'),
        })

        // Pending section should show the new file
        await expect(page.getByText('second.txt')).toBeVisible({ timeout: 5_000 })

        // Click Upload button to add the file
        await page.getByRole('button', { name: 'Upload', exact: true }).click()

        // Wait for the file to appear in the main file list
        await page.waitForTimeout(2_000)

        // Both files should be in the list now
        await expect(page.getByRole('link', { name: 'first.txt' })).toBeVisible()
        await expect(page.getByRole('link', { name: 'second.txt' })).toBeVisible({ timeout: 5_000 })
    })

    test('duplicate file names are skipped', async ({ page }) => {
        await uploadTestFile(page, 'unique.txt', 'unique content')

        // Try adding a file with the same name
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'unique.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('duplicate content'),
        })

        // The app should reject the duplicate — wait for any warning/toast
        // or verify the pending file count stays at 0
        await page.waitForTimeout(1_000)

        // Verify there's no pending file row (only the existing uploaded file)
        // The file should still just be in the uploaded list, no new pending items
        const pendingItems = page.locator('.pending-upload, [class*="pending"]')
        await expect(pendingItems).toHaveCount(0)
    })

    test('Add Files hidden for streaming uploads', async ({ page, withConfig }) => {
        await withConfig({ feature_stream: 'enabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Enable streaming toggle — the label contains 'Streaming' text and a .toggle-switch button
        const streamingLabel = page.locator('label').filter({ hasText: 'Streaming' })
        await streamingLabel.locator('.toggle-switch').click()

        // Add a file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'stream-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('stream content'),
        })

        // Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 15_000 })
        await page.waitForLoadState('networkidle')

        // Add Files should NOT be visible for streaming uploads
        await expect(page.getByRole('button', { name: /Add Files/i })).not.toBeVisible()
    })

    test('streaming upload shows info banner', async ({ page, withConfig }) => {
        await withConfig({ feature_stream: 'enabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const streamingLabel = page.locator('label').filter({ hasText: 'Streaming' })
        await streamingLabel.locator('.toggle-switch').click()

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'stream-banner.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('banner test'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 15_000 })
        await page.waitForLoadState('networkidle')

        // Streaming indicator banner should be visible
        await expect(page.getByText('Streaming Upload')).toBeVisible({ timeout: 5_000 })
        await expect(page.getByText(/share the upload link/i)).toBeVisible()
    })
})

test.describe('Delete file/upload', () => {
    test('delete file shows it as deleted', async ({ page }) => {
        // Upload two files so we can delete one
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const input = page.locator('input[type="file"]')
        await input.setInputFiles([
            { name: 'keep.txt', mimeType: 'text/plain', buffer: Buffer.from('keep me') },
            { name: 'remove.txt', mimeType: 'text/plain', buffer: Buffer.from('remove me') },
        ])

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Both files visible
        await expect(page.getByRole('link', { name: 'remove.txt' })).toBeVisible()

        // Click the remove button on the file (× button)
        const removeBtn = page.getByTitle('Remove file').last()
        await removeBtn.click()

        // Confirm dialog should appear
        const dialog = page.locator('.fixed.inset-0.z-50 .glass-card')
        await expect(dialog).toBeVisible({ timeout: 3_000 })
        await dialog.getByRole('button', { name: 'Delete' }).click()

        // Deleted file should still be visible but greyed-out with a Deleted badge
        await expect(page.getByText('Removed')).toBeVisible({ timeout: 5_000 })
        // The file name should be plain text (no download link), with line-through
        await expect(page.getByRole('link', { name: 'remove.txt' })).not.toBeVisible()
        await expect(page.getByText('remove.txt')).toBeVisible()
        // Other file still has its download link
        await expect(page.getByRole('link', { name: 'keep.txt' })).toBeVisible()
        // File count heading should say "1 file" (only the live one)
        await expect(page.getByRole('heading', { name: '1 file' })).toBeVisible({ timeout: 3_000 })
    })

    test('deleted file has no action buttons', async ({ page }) => {
        await uploadTestFile(page, 'action-test.txt', 'action test content')

        // Delete the file
        const removeBtn = page.getByTitle('Remove file').first()
        await removeBtn.click()
        const dialog = page.locator('.fixed.inset-0.z-50 .glass-card')
        await expect(dialog).toBeVisible({ timeout: 3_000 })
        await dialog.getByRole('button', { name: 'Delete' }).click()

        // Wait for the Deleted badge to appear
        await expect(page.getByText('Removed')).toBeVisible({ timeout: 5_000 })

        // No download, view, QR, copy, or remove buttons should be visible on the deleted row
        await expect(page.getByRole('link', { name: 'Download' })).not.toBeVisible()
        await expect(page.getByRole('button', { name: 'View' })).not.toBeVisible()
        await expect(page.getByTitle('Show QR code')).not.toBeVisible()
        await expect(page.getByTitle('Remove file')).not.toBeVisible()
    })

    test('delete upload redirects to home', async ({ page }) => {
        await uploadTestFile(page, 'delete-me.txt', 'delete content')

        // Click "Delete Upload" in sidebar
        const deleteUploadBtn = page.locator('aside').getByRole('button', { name: /Delete Upload/i })
        await deleteUploadBtn.click()

        // Confirm dialog
        const dialog = page.locator('.fixed.inset-0.z-50 .glass-card')
        await expect(dialog).toBeVisible({ timeout: 3_000 })
        await dialog.getByRole('button', { name: 'Delete' }).click()

        // Should redirect to home
        await page.waitForURL(/\/$|#\/$|#$/, { timeout: 5_000 })
    })

    test('delete file confirm can be cancelled', async ({ page }) => {
        await uploadTestFile(page, 'cancel-delete.txt', 'keep me')

        // Click remove button
        const removeBtn = page.getByTitle('Remove file').first()
        await removeBtn.click()

        // Confirm dialog appears
        await expect(page.getByText(/Delete File/i)).toBeVisible({ timeout: 3_000 })

        // Click Cancel
        await page.getByRole('button', { name: 'Cancel' }).click()

        // File still visible
        await expect(page.getByRole('link', { name: 'cancel-delete.txt' })).toBeVisible()
    })

    test('deleting viewed file closes viewer panel', async ({ page }) => {
        // Upload a single text file — the viewer auto-opens for single viewable files
        await uploadTestFile(page, 'viewed-file.txt', 'viewer content')
        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })

        // Delete the file via the remove button
        const removeBtn = page.getByTitle('Remove file').first()
        await removeBtn.click()

        // Confirm dialog
        const dialog = page.locator('.fixed.inset-0.z-50 .glass-card')
        await expect(dialog).toBeVisible({ timeout: 3_000 })
        await dialog.getByRole('button', { name: 'Delete' }).click()

        // Viewer panel should close
        await expect(panel).not.toBeVisible({ timeout: 5_000 })

        // Deleted file should still be visible with Deleted badge (not hidden)
        await expect(page.getByText('Removed')).toBeVisible({ timeout: 5_000 })
        await expect(page.getByText('viewed-file.txt')).toBeVisible()
    })
})

test.describe('Unauthenticated download permissions', () => {
    test('no Delete Upload button without token', async ({ page, context }) => {
        const url = await uploadTestFile(page, 'perm-test.txt', 'permission content')
        const uploadId = new URL(url.replace('#/', '')).searchParams.get('id')
            || new URLSearchParams(url.split('?')[1] || '').get('id')

        // Open a fresh page without upload token
        const freshPage = await context.newPage()
        await freshPage.goto(`/#/?id=${uploadId}`)
        await freshPage.waitForLoadState('networkidle')

        // Wait for the file to load
        await expect(freshPage.getByRole('link', { name: 'perm-test.txt' })).toBeVisible({ timeout: 5_000 })

        // Delete Upload button should NOT be visible
        await expect(freshPage.getByRole('button', { name: /Delete Upload/i })).not.toBeVisible()
        await freshPage.close()
    })

    test('no file remove buttons without token', async ({ page, context }) => {
        const url = await uploadTestFile(page, 'no-remove.txt', 'no remove')
        const uploadId = new URLSearchParams(url.split('?')[1] || '').get('id')

        const freshPage = await context.newPage()
        await freshPage.goto(`/#/?id=${uploadId}`)
        await freshPage.waitForLoadState('networkidle')
        await expect(freshPage.getByRole('link', { name: 'no-remove.txt' })).toBeVisible({ timeout: 5_000 })

        // Remove buttons should NOT be visible
        await expect(freshPage.getByTitle('Remove file')).not.toBeVisible()
        await freshPage.close()
    })

    test('no Add Files button without token', async ({ page, context }) => {
        const url = await uploadTestFile(page, 'no-add.txt', 'no add')
        const uploadId = new URLSearchParams(url.split('?')[1] || '').get('id')

        const freshPage = await context.newPage()
        await freshPage.goto(`/#/?id=${uploadId}`)
        await freshPage.waitForLoadState('networkidle')
        await expect(freshPage.getByRole('link', { name: 'no-add.txt' })).toBeVisible({ timeout: 5_000 })

        // Add Files should NOT be visible
        await expect(freshPage.getByRole('button', { name: /Add Files/i })).not.toBeVisible()
        await freshPage.close()
    })

    test('removable upload shows Delete button without token', async ({ page, context, withConfig }) => {
        await withConfig({ feature_removable: 'default' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Removable should be on by default with this config
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'removable.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('removable content'),
        })
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')
        const url = page.url()
        const uploadId = new URLSearchParams(url.split('?')[1] || '').get('id')

        // Open fresh page without token
        const freshPage = await context.newPage()
        await freshPage.goto(`/#/?id=${uploadId}`)
        await freshPage.waitForLoadState('networkidle')
        await expect(freshPage.getByRole('link', { name: 'removable.txt' })).toBeVisible({ timeout: 5_000 })

        // Delete Upload IS visible for removable uploads
        await expect(freshPage.getByRole('button', { name: /Delete Upload/i })).toBeVisible()
        await freshPage.close()
    })

    test('removable upload shows file remove buttons without token', async ({ page, context, withConfig }) => {
        await withConfig({ feature_removable: 'default' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'removable2.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('removable content 2'),
        })
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')
        const url = page.url()
        const uploadId = new URLSearchParams(url.split('?')[1] || '').get('id')

        const freshPage = await context.newPage()
        await freshPage.goto(`/#/?id=${uploadId}`)
        await freshPage.waitForLoadState('networkidle')
        await expect(freshPage.getByRole('link', { name: 'removable2.txt' })).toBeVisible({ timeout: 5_000 })

        // File remove buttons ARE visible for removable uploads
        await expect(freshPage.getByTitle('Remove file').first()).toBeVisible()
        await freshPage.close()
    })

    test('admin URL panel visible and link grants admin access', async ({ page, context }) => {
        await uploadTestFile(page, 'admin-access.txt', 'admin content')

        // Admin URL section should be visible in the sidebar
        const adminSection = page.locator('aside').getByText('Admin URL')
        await expect(adminSection).toBeVisible({ timeout: 5_000 })

        // Extract the admin URL text from the sidebar
        const adminUrlText = await page.locator('aside').locator('.sidebar-section').filter({ hasText: 'Admin URL' })
            .locator('.text-surface-300.truncate').textContent()
        expect(adminUrlText).toBeTruthy()
        expect(adminUrlText).toContain('uploadToken=')

        // Open the admin URL in a fresh page (no prior token in memory)
        const freshPage = await context.newPage()
        // Convert absolute URL to hash route if needed
        const hashPart = adminUrlText.includes('#') ? adminUrlText.substring(adminUrlText.indexOf('#')) : adminUrlText
        await freshPage.goto(hashPart)
        await freshPage.waitForLoadState('networkidle')

        // File should load
        await expect(freshPage.getByRole('link', { name: 'admin-access.txt' })).toBeVisible({ timeout: 5_000 })

        // Admin buttons should be visible (the token grants admin access)
        await expect(freshPage.getByRole('button', { name: /Delete Upload/i })).toBeVisible({ timeout: 3_000 })
        await expect(freshPage.getByRole('button', { name: /Add Files/i })).toBeVisible()
        await freshPage.close()
    })
})

test.describe('URL file selection', () => {
    test('file= query param opens the correct file viewer on load', async ({ page, context }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload 2 text files
        const input = page.locator('input[type="file"]')
        await input.setInputFiles([
            { name: 'alpha.txt', mimeType: 'text/plain', buffer: Buffer.from('alpha content') },
            { name: 'beta.txt', mimeType: 'text/plain', buffer: Buffer.from('beta content') },
        ])

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Both files listed
        await expect(page.getByRole('link', { name: 'alpha.txt' })).toBeVisible()
        await expect(page.getByRole('link', { name: 'beta.txt' })).toBeVisible()

        // Click View on second file to get its file ID from the URL
        const viewButtons = page.getByRole('button', { name: 'View', exact: true })
        await expect(viewButtons.nth(1)).toBeVisible({ timeout: 5_000 })
        await viewButtons.nth(1).click()

        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })

        // URL should now contain file= param
        const urlWithFile = page.url()
        expect(urlWithFile).toMatch(/file=/)

        // Extract upload ID and file ID from URL
        const hashPart = urlWithFile.substring(urlWithFile.indexOf('#'))
        const searchPart = hashPart.includes('?') ? hashPart.substring(hashPart.indexOf('?')) : ''
        const params = new URLSearchParams(searchPart)
        const uploadId = params.get('id')
        const fileId = params.get('file')
        expect(uploadId).toBeTruthy()
        expect(fileId).toBeTruthy()

        // Open in a fresh page with the file= param
        const freshPage = await context.newPage()
        await freshPage.goto(`/#/?id=${uploadId}&file=${fileId}`)
        await freshPage.waitForLoadState('networkidle')

        // Viewer should auto-open showing beta.txt
        const freshPanel = freshPage.locator('#file-viewer-panel')
        await expect(freshPanel).toBeVisible({ timeout: 5_000 })
        await expect(freshPanel).toContainText('beta.txt')
        await expect(freshPanel).toContainText('beta content')

        await freshPage.close()
    })

    test('URL updates with file= when viewer opens and clears when closed', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload 2 files so auto-view doesn't trigger
        const input = page.locator('input[type="file"]')
        await input.setInputFiles([
            { name: 'one.txt', mimeType: 'text/plain', buffer: Buffer.from('one') },
            { name: 'two.txt', mimeType: 'text/plain', buffer: Buffer.from('two') },
        ])

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Viewer should NOT be auto-open (multiple files)
        const panel = page.locator('#file-viewer-panel')
        await expect(panel).not.toBeVisible()

        // URL should NOT have file= param yet
        expect(page.url()).not.toMatch(/file=/)

        // Open viewer on first file
        const viewBtn = page.getByRole('button', { name: 'View', exact: true }).first()
        await expect(viewBtn).toBeVisible({ timeout: 5_000 })
        await viewBtn.click()
        await expect(panel).toBeVisible({ timeout: 5_000 })

        // URL should now have file= param
        expect(page.url()).toMatch(/file=/)

        // Close viewer
        await page.keyboard.press('Escape')
        await expect(panel).not.toBeVisible()

        // URL should no longer have file= param
        expect(page.url()).not.toMatch(/file=/)
    })

    test('viewer navigation updates file= in URL', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload 2 text files
        const input = page.locator('input[type="file"]')
        await input.setInputFiles([
            { name: 'first.txt', mimeType: 'text/plain', buffer: Buffer.from('first') },
            { name: 'second.txt', mimeType: 'text/plain', buffer: Buffer.from('second') },
        ])

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Open viewer on first file
        const viewBtn = page.getByRole('button', { name: 'View', exact: true }).first()
        await expect(viewBtn).toBeVisible({ timeout: 5_000 })
        await viewBtn.click()

        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })
        await expect(panel.locator('text=1/2')).toBeVisible()

        // Extract file= from hash URL
        const url1 = page.url()
        const hash1 = url1.substring(url1.indexOf('#'))
        const search1 = hash1.includes('?') ? hash1.substring(hash1.indexOf('?')) : ''
        const fileId1 = new URLSearchParams(search1).get('file')
        expect(fileId1).toBeTruthy()

        // Navigate to next file
        await page.keyboard.press('ArrowRight')
        await expect(panel.locator('text=2/2')).toBeVisible()

        // URL file= should point to a different file
        const url2 = page.url()
        const hash2 = url2.substring(url2.indexOf('#'))
        const search2 = hash2.includes('?') ? hash2.substring(hash2.indexOf('?')) : ''
        const fileId2 = new URLSearchParams(search2).get('file')
        expect(fileId2).toBeTruthy()
        expect(fileId2).not.toBe(fileId1)
    })
})

