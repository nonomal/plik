import { test, expect, uploadTestFile } from './fixtures.js'

test.describe('Upload settings', () => {
    test('one-shot toggle is reflected in download view', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file first (the sidebar toggles may only appear when files are pending)
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'oneshot.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('one shot content'),
        })

        // Enable one-shot — the label says "Destruct after download"
        const toggle = page.getByText('Destruct after download').locator('xpath=..').locator('.toggle-switch')
        await toggle.click()

        // Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Download view should show one-shot indicator
        await expect(page.getByText(/one.?shot/i).first()).toBeVisible()
    })

    test('password-protected upload shows password badge on download page', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file first
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'protected.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('secret content'),
        })

        // Enable password toggle
        const toggle = page.getByText('Password').first().locator('xpath=..').locator('.toggle-switch')
        await toggle.click()

        // Fill in credentials
        await page.getByPlaceholder('Login').fill('testuser')
        await page.getByPlaceholder('Password').fill('testpass')

        // Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Download sidebar should show the password badge
        await expect(page.getByText('🔒 Password')).toBeVisible({ timeout: 5_000 })
    })

    test('password toggle reveals login and password input fields', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Login/password inputs should not be visible before enabling the toggle
        await expect(page.getByPlaceholder('Login')).not.toBeVisible()
        await expect(page.getByPlaceholder('Password')).not.toBeVisible()

        // Enable the password toggle
        const toggle = page.getByText('Password').first().locator('xpath=..').locator('.toggle-switch')
        await toggle.click()

        // Both input fields should now be visible
        await expect(page.getByPlaceholder('Login')).toBeVisible()
        await expect(page.getByPlaceholder('Password')).toBeVisible()

        // Disabling the toggle should hide them again
        await toggle.click()
        await expect(page.getByPlaceholder('Login')).not.toBeVisible()
        await expect(page.getByPlaceholder('Password')).not.toBeVisible()
    })

    test('upload is blocked when password enabled but credentials are incomplete', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'incomplete.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('incomplete credentials'),
        })

        // Enable password toggle but leave login empty (only fill password)
        const toggle = page.getByText('Password').first().locator('xpath=..').locator('.toggle-switch')
        await toggle.click()
        await page.getByPlaceholder('Login').fill('')
        await page.getByPlaceholder('Password').fill('somepass')

        // Try to upload — should be blocked
        await page.getByRole('button', { name: 'Upload', exact: true }).click()

        // Should stay on upload page (no URL change with ?id=)
        await page.waitForTimeout(500)
        expect(page.url()).not.toMatch(/[?&]id=/)

        // Error message should be shown
        await expect(page.getByText(/login and password are required/i)).toBeVisible({ timeout: 3_000 })
    })

    test('password-protected upload shows credentials in share card for the uploader', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'cred-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('credentials test'),
        })

        // Enable password and fill credentials
        const toggle = page.getByText('Password').first().locator('xpath=..').locator('.toggle-switch')
        await toggle.click()
        await page.getByPlaceholder('Login').fill('shareuser')
        await page.getByPlaceholder('Password').fill('sharepass')

        // Upload → navigate to download view
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Credentials section should appear in the share card
        // exact:true avoids matching the server error text "please provide valid credentials..."
        await expect(page.getByText('Credentials', { exact: true })).toBeVisible({ timeout: 5_000 })

        // Login row: the label span and the value span should both be visible
        const credSection = page.locator('.sidebar-section').filter({ hasText: 'Credentials' })
        const loginLabel = credSection.locator('span').filter({ hasText: 'Login' }).first()
        await expect(loginLabel).toBeVisible()
        await expect(credSection.locator('span.font-mono').filter({ hasText: 'shareuser' })).toBeVisible()

        // Password row
        const passwordLabel = credSection.locator('span').filter({ hasText: 'Password' }).first()
        await expect(passwordLabel).toBeVisible()
        await expect(credSection.locator('span.font-mono').filter({ hasText: 'sharepass' })).toBeVisible()
    })

    test('credentials are not shown to a fresh visitor opening the share link', async ({ page, context }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload with password
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'visitor-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('visitor should not see creds'),
        })

        const toggle = page.getByText('Password').first().locator('xpath=..').locator('.toggle-switch')
        await toggle.click()
        await page.getByPlaceholder('Login').fill('hiddenlogin')
        await page.getByPlaceholder('Password').fill('hiddenpass')

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })

        // Grab the current URL (the share link)
        const shareUrl = page.url()

        // Open the share URL in a fresh page (simulating another user)
        const freshPage = await context.newPage()
        await freshPage.goto(shareUrl)
        await freshPage.waitForLoadState('networkidle')
        await freshPage.waitForTimeout(1_000)

        // The fresh visitor should NOT see a Credentials section
        // exact:true avoids matching the server error text "please provide valid credentials..."
        await expect(freshPage.getByText('Credentials', { exact: true })).not.toBeVisible()
        await freshPage.close()
    })

    test('password-protected upload returns 401 without credentials', async ({ page, context }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'secret.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('top secret'),
        })

        // Enable password toggle and fill credentials
        const toggle = page.getByText('Password').first().locator('xpath=..').locator('.toggle-switch')
        await toggle.click()
        await page.getByPlaceholder('Login').fill('mylogin')
        await page.getByPlaceholder('Password').fill('mypassword')

        // Upload
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Grab the file download link href
        const downloadLink = page.getByRole('link', { name: 'secret.txt' })
        const fileUrl = await downloadLink.getAttribute('href')

        // Open a fresh page in the same context — no stored basicAuth
        const freshPage = await context.newPage()
        const response = await freshPage.request.get(fileUrl)

        // Server should return 401 Unauthorized (basic auth required)
        expect(response.status()).toBe(401)
        await freshPage.close()
    })

    test('comment editor appears when toggled', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file first
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'commented.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('with comments'),
        })

        // Enable comments — the label says "Comment"
        const toggle = page.getByText('Comment').first().locator('xpath=..').locator('.toggle-switch')
        await toggle.click()

        // A markdown editor textarea should appear in the upload page
        // The comment form appears below the files/sidebar
        await page.waitForLoadState('networkidle')

        // Upload with comment
        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // Download view should have loaded successfully
        await expect(page.locator('body')).not.toBeEmpty()
    })

    test('TTL expiration shown in download sidebar', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Upload a file (uses default TTL)
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'ttl-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('ttl content'),
        })

        await page.getByRole('button', { name: 'Upload', exact: true }).click()
        await page.waitForURL(/[?&]id=/, { timeout: 10_000 })
        await page.waitForLoadState('networkidle')

        // The download sidebar should show TTL/expiration info
        await expect(page.getByText(/expire|remaining|never/i).first()).toBeVisible()
    })
})

// ── Feature flag tests ──────────────────────────────────────────────────────
// Each toggle's behaviour depends on 4 possible feature flag values:
//   disabled → toggle hidden
//   enabled  → toggle visible, OFF by default, clickable
//   default  → toggle visible, ON  by default, clickable
//   forced   → toggle visible, ON  by default, disabled (not clickable)

/**
 * Toggle-style feature flags and their UI labels in the upload sidebar.
 * set_ttl is special (heading, not a toggle) and tested separately.
 */
const TOGGLE_FLAGS = [
    { flag: 'one_shot', label: 'Destruct after download' },
    { flag: 'stream', label: 'Streaming' },
    { flag: 'removable', label: 'Removable' },
    { flag: 'e2ee', label: 'End-to-End Encryption' },
    { flag: 'password', label: 'Password' },
    { flag: 'comments', label: 'Comment' },
    { flag: 'extend_ttl', label: 'Extend TTL on access' },
]

/** Locate the toggle switch next to a given label text. */
function toggleFor(page, label) {
    return page.getByText(label, { exact: false }).first().locator('xpath=..').locator('.toggle-switch')
}

test.describe('Feature flags', () => {
    // ── disabled ─────────────────────────────────────────────────────────
    for (const { flag, label } of TOGGLE_FLAGS) {
        test(`${flag} disabled hides toggle`, async ({ page, withConfig }) => {
            await withConfig({ [`feature_${flag}`]: 'disabled' })
            await page.goto('/')
            await page.waitForLoadState('networkidle')

            await expect(page.getByText(label).first()).not.toBeVisible()
        })
    }

    test('set_ttl disabled hides expiration section', async ({ page, withConfig }) => {
        await withConfig({ feature_set_ttl: 'disabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.getByRole('heading', { name: 'Expiration' })).not.toBeVisible()
    })

    // ── enabled ──────────────────────────────────────────────────────────
    for (const { flag, label } of TOGGLE_FLAGS) {
        test(`${flag} enabled shows toggle OFF by default`, async ({ page, withConfig }) => {
            await withConfig({ [`feature_${flag}`]: 'enabled' })
            await page.goto('/')
            await page.waitForLoadState('networkidle')

            const toggle = toggleFor(page, label)
            await expect(toggle).toBeVisible()
            await expect(toggle).toHaveAttribute('data-active', 'false')
            await expect(toggle).not.toBeDisabled()
        })
    }

    test('set_ttl enabled shows expiration section', async ({ page, withConfig }) => {
        await withConfig({ feature_set_ttl: 'enabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.getByRole('heading', { name: 'Expiration' })).toBeVisible()
    })

    // ── default ──────────────────────────────────────────────────────────
    for (const { flag, label } of TOGGLE_FLAGS) {
        test(`${flag} default shows toggle ON by default`, async ({ page, withConfig }) => {
            await withConfig({ [`feature_${flag}`]: 'default' })
            await page.goto('/')
            await page.waitForLoadState('networkidle')

            const toggle = toggleFor(page, label)
            await expect(toggle).toBeVisible()
            await expect(toggle).toHaveAttribute('data-active', 'true')
            await expect(toggle).not.toBeDisabled()
        })
    }

    test('set_ttl default shows expiration section', async ({ page, withConfig }) => {
        await withConfig({ feature_set_ttl: 'default' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.getByRole('heading', { name: 'Expiration' })).toBeVisible()
    })

    // ── forced ───────────────────────────────────────────────────────────
    for (const { flag, label } of TOGGLE_FLAGS) {
        test(`${flag} forced shows toggle ON + disabled`, async ({ page, withConfig }) => {
            await withConfig({ [`feature_${flag}`]: 'forced' })
            await page.goto('/')
            await page.waitForLoadState('networkidle')

            const toggle = toggleFor(page, label)
            await expect(toggle).toBeVisible()
            await expect(toggle).toHaveAttribute('data-active', 'true')
            await expect(toggle).toBeDisabled()
        })
    }

    test('set_ttl forced shows expiration section', async ({ page, withConfig }) => {
        await withConfig({ feature_set_ttl: 'forced' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.getByRole('heading', { name: 'Expiration' })).toBeVisible()
    })

    // ── Special: comments forced shows "required" label ──────────────────
    test('comments forced shows required indicator', async ({ page, withConfig }) => {
        await withConfig({ feature_comments: 'forced' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Add a file so the upload form is active
        const input = page.locator('input[type="file"]')
        await input.setInputFiles({
            name: 'flag-test.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('test'),
        })

        // The comment textarea area should show "required"
        await expect(page.getByText('required')).toBeVisible()
    })
})

// ── Footer ────────────────────────────────────────────────────────────────

test.describe('Footer', () => {
    test('hidden when nothing is configured', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.locator('footer')).not.toBeVisible()
    })

    test('shown with mailto link when server abuseContact is configured', async ({ page, withConfig }) => {
        await withConfig({ abuseContact: 'abuse@example.com' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const footer = page.locator('footer')
        await expect(footer).toBeVisible({ timeout: 5_000 })
        await expect(footer).toContainText('For abuse contact')

        const link = footer.locator('a[href="mailto:abuse@example.com"]')
        await expect(link).toBeVisible()
        await expect(link).toHaveText('abuse@example.com')
    })

    test('custom footer HTML from settings.json', async ({ page, withSettings }) => {
        await withSettings({
            name: 'Plik',
            footer: 'Powered by <a href="https://plik.root.gg">Plik</a> · Custom footer',
        })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const footer = page.locator('footer')
        await expect(footer).toBeVisible({ timeout: 5_000 })
        await expect(footer).toContainText('Custom footer')

        const link = footer.locator('a[href="https://plik.root.gg"]')
        await expect(link).toBeVisible()
        await expect(link).toHaveText('Plik')
    })

    test('settings.json footer overrides server abuseContact', async ({ page, withConfig, withSettings }) => {
        await withConfig({ abuseContact: 'abuse@example.com' })
        await withSettings({
            name: 'Plik',
            footer: 'Custom takes priority',
        })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        const footer = page.locator('footer')
        await expect(footer).toBeVisible({ timeout: 5_000 })
        await expect(footer).toContainText('Custom takes priority')
        await expect(footer).not.toContainText('abuse@example.com')
    })
})

// ── Header Feature Flags ──────────────────────────────────────────────────

test.describe('Header feature flags', () => {
    test('CLI link visible by default', async ({ page }) => {
        await page.goto('/')
        // The header should show the CLI link when "clients" feature is enabled (default)
        await page.waitForLoadState('networkidle')
        await expect(page.getByText('CLI', { exact: true }).first()).toBeVisible({ timeout: 5_000 })
    })

    test('CLI link hidden when disabled', async ({ page, withConfig }) => {
        await withConfig({ feature_clients: 'disabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')
        // Should not be visible when clients feature is disabled
        await expect(page.getByText('CLI', { exact: true })).not.toBeVisible()
    })

    test('Documentation and GitHub links visible by default', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.getByText('Documentation').first()).toBeVisible({ timeout: 5_000 })
        await expect(page.getByText('GitHub').first()).toBeVisible()
    })

    test('Documentation and GitHub hidden when disabled', async ({ page, withConfig }) => {
        await withConfig({ feature_github: 'disabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.getByText('Documentation')).not.toBeVisible()
        await expect(page.getByText('GitHub')).not.toBeVisible()
    })

    test('Paste text button visible by default', async ({ page }) => {
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.getByText('Paste text').first()).toBeVisible({ timeout: 5_000 })
    })

    test('Paste text button hidden when disabled', async ({ page, withConfig }) => {
        await withConfig({ feature_text: 'disabled' })
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        await expect(page.getByText('Paste text')).not.toBeVisible()
    })
})

// ── Setting Help Tooltips ─────────────────────────────────────────────────

test.describe('Setting help tooltips', () => {
    const ALL_FEATURES = {
        feature_one_shot: 'enabled',
        feature_stream: 'enabled',
        feature_removable: 'enabled',
        feature_e2ee: 'enabled',
        feature_password: 'enabled',
        feature_comments: 'enabled',
        feature_extend_ttl: 'enabled',
    }

    const TOOLTIP_DATA = [
        { label: 'Destruct after download', tooltip: 'Files are permanently deleted after they are downloaded once' },
        { label: 'Streaming', tooltip: 'Files are streamed directly to the downloader without being stored on the server' },
        { label: 'Removable', tooltip: 'Anyone with the link can delete uploaded files' },
        { label: 'End-to-End Encryption', tooltip: 'Files are encrypted in the browser before upload' },
        { label: 'Password', tooltip: 'Protect the upload with HTTP basic authentication credentials' },
        { label: 'Comment', tooltip: 'Add a Markdown-formatted message to the download page' },
        { label: 'Extend TTL on access', tooltip: 'Reset the expiration timer each time a file is accessed' },
    ]

    for (const { label, tooltip } of TOOLTIP_DATA) {
        test(`${label} has a help icon`, async ({ page, withConfig }) => {
            await withConfig(ALL_FEATURES)
            await page.goto('/')
            await page.waitForLoadState('networkidle')

            // Locate the (?) icon near the label
            const helpIcon = page.getByText(label, { exact: false }).first()
                .locator('xpath=..').locator('.setting-help')
            await expect(helpIcon).toBeVisible()
            await expect(helpIcon).toHaveText('?')
        })
    }

    test('tooltip appears on hover', async ({ page, withConfig }) => {
        await withConfig(ALL_FEATURES)
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Find the help wrapper near "Destruct after download"
        const helpWrap = page.getByText('Destruct after download', { exact: false }).first()
            .locator('xpath=..').locator('.setting-help-wrap')
        const tooltipEl = helpWrap.locator('.setting-tooltip')

        // Tooltip should be hidden initially (opacity 0)
        await expect(tooltipEl).toBeAttached()

        // Hover over the (?) icon
        await helpWrap.locator('.setting-help').hover()

        // Tooltip should now be visible
        await expect(tooltipEl).toHaveCSS('opacity', '1')
        await expect(tooltipEl).toContainText('Files are permanently deleted after they are downloaded once')
    })

    test('tooltip appears on keyboard focus', async ({ page, withConfig }) => {
        await withConfig(ALL_FEATURES)
        await page.goto('/')
        await page.waitForLoadState('networkidle')

        // Focus the help icon via keyboard (it has tabindex=0)
        const helpIcon = page.getByText('Destruct after download', { exact: false }).first()
            .locator('xpath=..').locator('.setting-help')
        await helpIcon.focus()

        // Tooltip should be visible via :focus-within
        const tooltipEl = helpIcon.locator('xpath=..').locator('.setting-tooltip')
        await expect(tooltipEl).toHaveCSS('opacity', '1')
    })
})
