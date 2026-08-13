import { test, expect } from './fixtures.js'

test.describe('Language Picker — visibility', () => {
    test('picker is visible when multiple languages are available (default: all built-ins)', async ({ page, withLanguages }) => {
        await withLanguages(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#language-picker-toggle')
        await expect(picker).toBeVisible()
    })

    test('picker is hidden when only one language is configured', async ({ page, withLanguages }) => {
        await withLanguages(['en'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#language-picker-toggle')
        await expect(picker).toHaveCount(0)
    })

    test('picker is hidden when only Auto is in the list', async ({ page, withLanguages }) => {
        await withLanguages(['auto'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#language-picker-toggle')
        await expect(picker).toHaveCount(0)
    })

    test('picker shows when exactly two languages configured', async ({ page, withLanguages }) => {
        await withLanguages(['en', 'fr'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#language-picker-toggle')
        await expect(picker).toBeVisible()
    })
})

test.describe('Language Picker — dropdown', () => {
    test('clicking opens dropdown with language options', async ({ page, withLanguages }) => {
        await withLanguages(['en', 'fr', 'de'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Dropdown should not be visible initially
        await expect(page.locator('#lang-option-en')).toHaveCount(0)

        // Click the picker toggle
        await page.locator('#language-picker-toggle').click()

        // Dropdown should now show the configured languages
        await expect(page.locator('#lang-option-en')).toBeVisible()
        await expect(page.locator('#lang-option-fr')).toBeVisible()
        await expect(page.locator('#lang-option-de')).toBeVisible()

        // Languages NOT in the list should not appear
        await expect(page.locator('#lang-option-zh')).toHaveCount(0)
    })

    test('clicking outside closes the dropdown', async ({ page, withLanguages }) => {
        await withLanguages(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#language-picker-toggle').click()
        await expect(page.locator('#lang-option-en')).toBeVisible()

        // Click outside (on the body)
        await page.locator('body').click({ position: { x: 10, y: 300 } })

        // Dropdown should close
        await expect(page.locator('#lang-option-en')).toHaveCount(0)
    })

    test('selecting a language closes the dropdown', async ({ page, withLanguages }) => {
        await withLanguages(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#language-picker-toggle').click()
        await page.locator('#lang-option-fr').click()

        // Dropdown should close after selection
        await expect(page.locator('#lang-option-fr')).toHaveCount(0)
    })

    test('dropdown has scrollbar when many items', async ({ page, withLanguages }) => {
        await withLanguages(["*"])  // all built-in languages
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#language-picker-toggle').click()

        // The dropdown list container should have max-h-80 + overflow-y-auto
        const list = page.locator('.max-h-80.overflow-y-auto').filter({ has: page.locator('#lang-option-en') })
        await expect(list).toBeVisible()
    })
})

test.describe('Language Picker — language application', () => {
    test('selecting a language updates localStorage', async ({ page, withLanguages }) => {
        await withLanguages(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#language-picker-toggle').click()
        await page.locator('#lang-option-fr').click()

        const stored = await page.evaluate(() => localStorage.getItem('plik-locale'))
        expect(stored).toBe('fr')
    })

    test('active language shows checkmark', async ({ page, withLanguages }) => {
        await withLanguages(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Open picker and select French
        await page.locator('#language-picker-toggle').click()
        await page.locator('#lang-option-fr').click()

        // Reopen picker — fr should have the accent color (checkmark)
        await page.locator('#language-picker-toggle').click()
        const frOption = page.locator('#lang-option-fr')
        await expect(frOption).toHaveClass(/text-accent-400/)
    })
})

test.describe('Language Picker — localStorage persistence', () => {
    test('selected language persists across page reload', async ({ page, withLanguages }) => {
        await withLanguages(["*"])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Select French
        await page.locator('#language-picker-toggle').click()
        await page.locator('#lang-option-fr').click()

        // Verify localStorage was set
        const stored = await page.evaluate(() => localStorage.getItem('plik-locale'))
        expect(stored).toBe('fr')

        // Reload the page (route intercept persists)
        await page.reload({ waitUntil: 'networkidle' })

        // Language should still be fr in localStorage
        const storedAfter = await page.evaluate(() => localStorage.getItem('plik-locale'))
        expect(storedAfter).toBe('fr')
    })

    test('stale localStorage is ignored when language is not in available list', async ({ page, withLanguages }) => {
        // Pre-seed localStorage with a language not in the restricted list
        await page.addInitScript(() => localStorage.setItem('plik-locale', 'zh'))

        await withLanguages(['en', 'fr'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Open picker — zh should NOT be in the list
        await page.locator('#language-picker-toggle').click()
        await expect(page.locator('#lang-option-zh')).toHaveCount(0)
    })
})

test.describe('Language Picker — wildcard languages', () => {
    test('"*" expands to all built-in languages', async ({ page, withLanguages }) => {
        await withLanguages(['*'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#language-picker-toggle').click()

        // Built-in languages should be present
        await expect(page.locator('#lang-option-auto')).toBeVisible()
        await expect(page.locator('#lang-option-en')).toBeVisible()
        await expect(page.locator('#lang-option-fr')).toBeVisible()
        await expect(page.locator('#lang-option-de')).toBeVisible()
        await expect(page.locator('#lang-option-zh')).toBeVisible()
    })
})

test.describe('Language Picker — layout', () => {
    test('has "Language" text label in the button', async ({ page, withLanguages }) => {
        await withLanguages(["*"])
        // Use wider viewport so the desktop nav is visible
        await page.setViewportSize({ width: 1280, height: 720 })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const picker = page.locator('#language-picker-toggle')
        await expect(picker).toBeVisible()
        await expect(picker).toContainText('Language')
    })

    test('language options show flag images', async ({ page, withLanguages }) => {
        await withLanguages(['en', 'fr'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await page.locator('#language-picker-toggle').click()

        // The fr option should contain a flag image
        const frOption = page.locator('#lang-option-fr')
        await expect(frOption).toBeVisible()
        const flagImg = frOption.locator('img')
        await expect(flagImg).toHaveCount(1)
    })
})

test.describe('Language Picker — ordering', () => {
    test('languages are listed in alphabetical order by code (auto first)', async ({ page, withLanguages }) => {
        await withLanguages(['*'])
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Open the language picker dropdown
        await page.locator('#language-picker-toggle').click()

        // Collect all language option IDs in DOM order
        const items = page.locator('[id^="lang-option-"]')
        await expect(items.first()).toBeVisible({ timeout: 3_000 })

        const ids = await items.evaluateAll(els =>
            els.map(el => el.id.replace('lang-option-', ''))
        )

        // 'auto' must be first
        expect(ids[0]).toBe('auto')

        // The rest must be in alphabetical order
        const langCodes = ids.slice(1)
        const sorted = [...langCodes].sort()
        expect(langCodes).toEqual(sorted)
    })
})
