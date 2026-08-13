# Internationalization (i18n)

Plik supports multiple languages in the web interface, with automatic detection and user preference persistence.

## How It Works

1. **Auto-detection**: By default, Plik detects the user's preferred language from their browser settings
2. **Local storage**: Once a language is selected, it's saved to the browser's localStorage
3. **Server sync**: For authenticated users, the language preference is saved to their account and follows them across devices

## Built-in Languages

| Code | Language |
|------|----------|
| `de` | Deutsch |
| `en` | English |
| `es` | Español |
| `fr` | Français |
| `hi` | हिन्दी |
| `it` | Italiano |
| `nl` | Nederlands |
| `pl` | Polski |
| `pt` | Português |
| `ru` | Русский |
| `sv` | Svenska |
| `zh` | 中文 |

## Configuration

Language settings are configured in [`settings.json`](./web-ui#settings):

```jsonc
{
  // Default language: "auto" (detect from browser), "en", "fr", etc.
  "language": "auto",

  // Available languages in the picker ("*" = all built-in, [] = English only)
  "languages": ["*"]
}
```

### Language Picker

Users can switch languages via the globe icon in the header. The selected language persists across page reloads via localStorage. For authenticated users, the preference is also saved to their account — it follows them across devices and browsers automatically.

To control which languages appear in the picker, set the `languages` array in `settings.json`:

```jsonc
// All built-in languages (default)
"languages": ["*"]

// No language picker (English only)
"languages": []

// Only specific languages
"languages": ["auto", "en", "fr"]
```

When only one language is configured, the picker is hidden automatically.

## Adding a New Language

Follow these steps to add a new language to Plik:

### 1. Create the Locale File

Copy the English locale as a template:

```bash
cp webapp/src/locales/en.json webapp/src/locales/XX.json
```

Replace `XX` with the [ISO 639-1 language code](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) (e.g., `de` for German, `es` for Spanish).

### 2. Translate All Keys

Open your new `XX.json` file and translate all values. Keep the JSON keys unchanged — only translate the values.

::: tip Punctuation
Different languages have different punctuation rules. For example, French uses a space before colons (` :`), while English does not. Include any language-specific punctuation directly in the translation values.
:::

### 3. Register the Language

Add a flag SVG to `webapp/public/flags/`:

```bash
# Use any SVG flag — e.g. from https://github.com/lipis/flag-icons
cp your-flag.svg webapp/public/flags/XX.svg
```

Then edit `webapp/src/settings.js` and add your language to the `BUILTIN_LANGUAGES` array:

```javascript
export const BUILTIN_LANGUAGES = [
    { name: 'auto', label: 'Auto' },
    { name: 'en', label: 'English', flag: '/flags/en.svg' },
    { name: 'fr', label: 'Français', flag: '/flags/fr.svg' },
    // Add your language:
    { name: 'XX', label: 'Your Language', flag: '/flags/XX.svg' },
]
```

The `flag` field is a path to an SVG file in `webapp/public/flags/`. This allows custom flags (e.g. regional languages) that aren't in any icon library.

### 4. Import the Locale

Edit `webapp/src/i18n.js` and add the import:

```javascript
import en from './locales/en.json'
import fr from './locales/fr.json'
import XX from './locales/XX.json'  // [!code ++]

const i18n = createI18n({
    // ...
    messages: { en, fr, XX },  // [!code ++]
})
```

### 5. Build and Test

```bash
cd webapp
npm test        # Run unit tests
npm run build   # Verify production build
```

### 6. Configure (Optional)

If you want your language available in the picker by default, the `["*"]` wildcard in `settings.json` will automatically include it. For custom deployments, add it explicitly:

```jsonc
"languages": ["XX"]
```

## Internationalized Customizations

Runtime customizations like the `footer` in `settings.json` are static HTML — they don't change when the user switches language. To make them language-aware, use `customJS` with a [`MutationObserver`](https://developer.mozilla.org/en-US/docs/Web/API/MutationObserver) that reacts to locale changes.

### How It Works

When the user switches language, Plik sets `document.documentElement.lang` to the new locale (e.g. `"fr"`, `"de"`). A custom script can observe this attribute and swap content accordingly.

### Example: Internationalized Footer

**`settings.json`**:

```jsonc
{
  "customJS": "/js/custom.js",
  // Fallback shown before custom.js loads
  "footer": "For abuse contact <a href='mailto:abuse@example.com'>abuse@example.com</a>"
}
```

**`js/custom.js`**:

```javascript
;(function () {
    const link = '<a href="mailto:abuse@example.com" class="underline hover:text-surface-200">abuse@example.com</a>'

    // Translated footer per locale — add new languages here
    const footers = {
        en: `For abuse contact ${link}`,
        fr: `Pour signaler un abus, contactez ${link}`,
        de: `Bei Missbrauch kontaktieren Sie ${link}`,
        hi: `दुरुपयोग की शिकायत के लिए ${link} से संपर्क करें`,
        es: `Para reportar abuso, contacte ${link}`,
        it: `Per segnalare un abuso, contattare ${link}`,
        pt: `Para denunciar abuso, contacte ${link}`,
        nl: `Voor misbruik neem contact op met ${link}`,
        pl: `Aby zgłosić nadużycie, skontaktuj się z ${link}`,
        zh: `如需举报滥用行为，请联系 ${link}`,
        ru: `Для сообщения о нарушениях свяжитесь с ${link}`,
    }

    function updateFooter() {
        const lang = document.documentElement.lang || 'en'
        const el = document.querySelector('footer')
        if (el) el.innerHTML = footers[lang] || footers.en
    }

    // Re-translate footer whenever the user switches language (lives for the page lifetime)
    const langObserver = new MutationObserver(updateFooter)
    langObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['lang'] })

    // First run — wait for Vue to render the <footer> element
    const domObserver = new MutationObserver(() => {
        if (document.querySelector('footer')) {
            updateFooter()
            domObserver.disconnect()
        }
    })
    domObserver.observe(document.body, { childList: true, subtree: true })
})()
```

::: tip Docker
Mount `custom.js` alongside `settings.json`:
```bash
docker run -p 8080:8080 \
  -v ./settings.json:/home/plik/server/webapp/dist/settings.json:ro \
  -v ./custom.js:/home/plik/server/webapp/dist/js/custom.js:ro \
  rootgg/plik
```
:::

This same pattern works for any element — not just the footer. Observe `document.documentElement.lang` and swap content based on the active locale.

## Known Limitations

::: info Server-Side Errors
Error messages returned by the Plik server (e.g. "Invalid credentials", "Upload not found") are currently displayed in English regardless of the selected language. Only the webapp's own UI labels, buttons, and client-side error messages are fully translated. Server-side internationalization may be added in a future release.
:::

## Agent Workflows

If you're using an AI coding assistant, these workflows automate the i18n process:

- **`/add-language`** — End-to-end workflow: create locale file, flag SVG, register in settings/i18n, update all languagePicker sections, run tests, update docs
- **`/review-language`** — Quality review: automated key sync, loanword audit, punctuation rules, plural form validation, contextual spot-checks
