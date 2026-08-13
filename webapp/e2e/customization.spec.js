import { test, expect } from './fixtures.js'

test.describe('Customization — settings.json', () => {
    test('default settings: logo text and page title are "Plik"', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Logo text should be "Plik" (from the default settings.json)
        const logo = page.locator('.plik-logo-text').first()
        await expect(logo).toHaveText('Plik')

        // Page title should be "Plik"
        await expect(page).toHaveTitle('Plik')
    })

    test('custom name via settings.json override', async ({ page, withSettings }) => {
        await withSettings({
            name: 'MyFileShare',
            backgroundImage: '',
            backgroundColor: '',
            overlayOpacity: 0.2,
            customCSS: '',
            customJS: '',
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Logo text should be the custom name
        const logo = page.locator('.plik-logo-text').first()
        await expect(logo).toHaveText('MyFileShare')

        // Page title should match
        await expect(page).toHaveTitle('MyFileShare')
    })

    test('missing settings.json falls back to empty name (white-label safe)', async ({ page, withSettings }) => {
        await withSettings(null)

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Logo text should be empty (never leaks "Plik")
        const logo = page.locator('.plik-logo-text').first()
        await expect(logo).toHaveText('')

        // Page title should be empty
        await expect(page).toHaveTitle('')
    })
})

test.describe('Customization — theme', () => {
    test('auto theme follows OS dark preference', async ({ page }) => {
        // Emulate dark color scheme
        await page.emulateMedia({ colorScheme: 'dark' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).toBe('dark')
    })

    test('auto theme follows OS light preference', async ({ page }) => {
        await page.emulateMedia({ colorScheme: 'light' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).toBe('light')
    })

    test('explicit dark override ignores OS light preference', async ({ page, withSettings }) => {
        await page.emulateMedia({ colorScheme: 'light' })
        await withSettings({ name: 'Plik', theme: 'dark' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).toBe('dark')
    })

    test('explicit light override ignores OS dark preference', async ({ page, withSettings }) => {
        await page.emulateMedia({ colorScheme: 'dark' })
        await withSettings({ name: 'Plik', theme: 'light' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).toBe('light')
    })

    test('custom theme loads CSS file and sets data-theme', async ({ page, withSettings }) => {
        await withSettings({ name: 'Plik', theme: 'solarized-dark' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // data-theme should match the custom theme name
        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).toBe('solarized-dark')

        // The theme CSS file should have been injected as a <link> tag
        const link = page.locator('link[href*="solarized-dark.css"]')
        await expect(link).toHaveCount(1)
    })

    test('non-existent theme falls back gracefully', async ({ page, withSettings }) => {
        await withSettings({ name: 'Plik', theme: 'nonexistent' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // The theme system validates against available themes, so "nonexistent"
        // falls back to the first available theme (auto → resolves to dark/light).
        const theme = await page.evaluate(() => document.documentElement.dataset.theme)
        expect(theme).not.toBe('nonexistent')
        expect(['dark', 'light']).toContain(theme)

        // Page should still be visible and functional
        await expect(page.locator('#app')).toBeVisible()
    })
})

test.describe('Customization — logo', () => {
    test('default settings: no logo image rendered', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // No <img> logo should be present
        await expect(page.locator('.plik-logo-img')).toHaveCount(0)

        // Text logo should be visible
        const logo = page.locator('.plik-logo-text').first()
        await expect(logo).toBeVisible()
    })

    test('custom logo image via settings.json override', async ({ page, withSettings }) => {
        await withSettings({ name: 'MyApp', logo: '/img/test-logo.png' })

        // Serve a 1x1 transparent PNG so the <img> loads
        await page.route('**/img/test-logo.png', async (route) => {
            const pixel = Buffer.from(
                'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVQI12NgAAIABQABNjN9GQAAAAlwSFlzAAAWJQAAFiUBSVIk8AAAAA0lEQVQI12P4z8BQDwAEgAF/QualzQAAAABJRU5ErkJggg==',
                'base64'
            )
            await route.fulfill({
                status: 200,
                contentType: 'image/png',
                body: pixel,
            })
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Image logo should be rendered
        const img = page.locator('.plik-logo-img').first()
        await expect(img).toBeVisible()
        await expect(img).toHaveAttribute('src', '/img/test-logo.png')
        await expect(img).toHaveAttribute('alt', 'MyApp')

        // Text logo should NOT be rendered
        await expect(page.locator('.plik-logo-text')).toHaveCount(0)

        // Page title should still use the name
        await expect(page).toHaveTitle('MyApp')
    })
})

test.describe('Customization — custom CSS', () => {
    test('custom CSS is injected when customCSS is set', async ({ page, withSettings }) => {
        await withSettings({ name: 'Plik', customCSS: '/css/test-custom.css', customJS: '' })

        // Serve the custom CSS that sets a distinctive background color
        await page.route('**/css/test-custom.css', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'text/css',
                body: 'body { background-color: rgb(255, 0, 255) !important; }',
            })
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Verify the custom CSS was applied
        const bgColor = await page.evaluate(() => {
            return window.getComputedStyle(document.body).backgroundColor
        })
        expect(bgColor).toBe('rgb(255, 0, 255)')
    })

    test('no CSS injected when customCSS is empty', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // No custom stylesheet should be injected (only the app's own styles)
        const customLinks = await page.evaluate(() => {
            return Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
                .filter(link => link.href.includes('custom'))
                .length
        })
        expect(customLinks).toBe(0)
    })
})

test.describe('Customization — custom JS', () => {
    test('custom JS is injected when customJS is set', async ({ page, withSettings }) => {
        await withSettings({ name: 'Plik', customCSS: '', customJS: '/js/test-custom.js' })

        // Serve a custom JS that sets a global marker
        await page.route('**/js/test-custom.js', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/javascript',
                body: 'window.__CUSTOM_JS_LOADED__ = true;',
            })
        })

        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Verify the custom script was executed
        const loaded = await page.evaluate(() => window.__CUSTOM_JS_LOADED__)
        expect(loaded).toBe(true)
    })
})
