import { test, expect, uploadTestFile } from './fixtures.js'

test.describe('Mermaid diagram rendering', () => {
    const MERMAID_MD = '# Diagram\n\n```mermaid\ngraph TD\n    A[Start] --> B{Check}\n    B -->|Yes| C[Done]\n    B -->|No| D[Retry]\n```\n'

    test('renders mermaid diagram as SVG in file preview', async ({ page }) => {
        await uploadTestFile(page, 'diagram.md', MERMAID_MD)

        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })

        // Mermaid should render as SVG
        await expect(panel.locator('.mermaid svg')).toBeVisible({ timeout: 10_000 })

        // Raw source text should NOT be visible (mermaid replaces it with SVG)
        await expect(panel.locator('.mermaid')).not.toContainText('graph TD')
    })

    test('preserves source in data-source attribute', async ({ page }) => {
        await uploadTestFile(page, 'source.md', MERMAID_MD)

        const panel = page.locator('#file-viewer-panel')
        await expect(panel.locator('.mermaid svg')).toBeVisible({ timeout: 10_000 })

        // data-source should contain the original diagram text
        const source = await panel.locator('.mermaid').getAttribute('data-source')
        expect(source).toContain('graph TD')
        expect(source).toContain('A[Start]')
    })

    test('renders multiple mermaid diagrams in a single file', async ({ page }) => {
        const multiDiagram = [
            '# Two diagrams',
            '',
            '```mermaid',
            'graph LR',
            '    A --> B',
            '```',
            '',
            'Some text in between.',
            '',
            '```mermaid',
            'sequenceDiagram',
            '    Alice->>Bob: Hello',
            '```',
        ].join('\n')

        await uploadTestFile(page, 'multi.md', multiDiagram)

        const panel = page.locator('#file-viewer-panel')
        await expect(panel).toBeVisible({ timeout: 5_000 })

        // Both diagrams should render as SVGs
        const svgs = panel.locator('.mermaid svg')
        await expect(svgs).toHaveCount(2, { timeout: 10_000 })
    })

    test('mermaid in upload comment renders as SVG', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'commented.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('file with mermaid comment'),
        })

        // Enable comment and write mermaid content
        const toggle = page.getByText('Comment').first().locator('xpath=..').locator('.toggle-switch')
        await toggle.click()
        await page.locator('textarea').fill('# Flow\n\n```mermaid\ngraph TD\n    X --> Y\n```')

        // Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Comment section heading should be visible (confirms markdown rendered)
        await expect(page.getByRole('heading', { name: 'Comment' })).toBeVisible({ timeout: 5_000 })
        await expect(page.locator('.prose h1')).toHaveText('Flow')

        // Mermaid diagram in the comment should render as SVG
        const commentSection = page.locator('.prose').filter({ has: page.locator('.mermaid') })
        await expect(commentSection.locator('.mermaid svg')).toBeVisible({ timeout: 15_000 })
    })
})

test.describe('Mermaid theme reactivity', () => {
    const MERMAID_MD = '```mermaid\ngraph TD\n    A --> B\n```\n'

    test('re-renders mermaid diagrams when theme changes', async ({ page, withThemes }) => {
        await withThemes(['dark', 'light'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload a markdown file with a mermaid diagram
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'theme-test.md',
            mimeType: 'text/plain',
            buffer: Buffer.from(MERMAID_MD),
        })
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        const panel = page.locator('#file-viewer-panel')
        await expect(panel.locator('.mermaid svg')).toBeVisible({ timeout: 10_000 })

        // Capture the initial SVG content for comparison
        const initialSvg = await panel.locator('.mermaid svg').innerHTML()

        // Switch from dark to light theme
        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-light').click()
        await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')

        // Wait for the diagram to re-render (SVG should still be present)
        await expect(panel.locator('.mermaid svg')).toBeVisible({ timeout: 10_000 })

        // The SVG content should differ (different theme colors)
        const newSvg = await panel.locator('.mermaid svg').innerHTML()
        expect(newSvg).not.toBe(initialSvg)
    })

    test('mermaid diagram still renders after switching back and forth', async ({ page, withThemes }) => {
        await withThemes(['dark', 'light'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'bounce.md',
            mimeType: 'text/plain',
            buffer: Buffer.from(MERMAID_MD),
        })
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        const panel = page.locator('#file-viewer-panel')
        await expect(panel.locator('.mermaid svg')).toBeVisible({ timeout: 10_000 })

        // Dark → Light
        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-light').click()
        await expect(panel.locator('.mermaid svg')).toBeVisible({ timeout: 10_000 })

        // Light → Dark
        await page.locator('#theme-picker-toggle').click()
        await page.locator('#theme-option-dark').click()
        await expect(panel.locator('.mermaid svg')).toBeVisible({ timeout: 10_000 })

        // Raw source should never leak
        await expect(panel.locator('.mermaid')).not.toContainText('graph TD')
    })
})
