import { marked } from 'marked'
import DOMPurify from 'dompurify'

/**
 * Custom renderer that converts ```mermaid code blocks into
 * <div class="mermaid"> containers (instead of <pre><code>).
 * Mermaid.run() will later transform them into SVGs.
 */
const renderer = {
    code({ text, lang }) {
        if (lang === 'mermaid') {
            return `<div class="mermaid">${text}</div>\n`
        }
        // Fall through to default marked behaviour
        return false
    },
}

marked.use({ renderer })

/**
 * Render Markdown text to sanitized HTML.
 *
 * Uses DOMPurify to prevent XSS from user-supplied content
 * (e.g. upload comments rendered via v-html).
 *
 * @param {string} text - Raw Markdown text
 * @returns {string} Sanitized HTML string
 */
export function renderMarkdown(text) {
    if (!text) return ''
    const html = marked.parse(text, { breaks: true })
    return DOMPurify.sanitize(html)
}

/**
 * Detect the mermaid theme from the current document color scheme.
 */
function getMermaidTheme() {
    const isDark = getComputedStyle(document.documentElement).colorScheme !== 'light'
    return isDark ? 'dark' : 'default'
}

/**
 * Lazy-load Mermaid and render all `.mermaid` divs inside a container.
 *
 * Call this AFTER Vue has injected rendered HTML via v-html + nextTick
 * so the DOM nodes actually exist.
 *
 * `mermaid.initialize()` runs once (guarded by `mermaidReady`).
 * Original diagram source is stashed in `data-source` so diagrams can
 * be re-rendered when the theme changes at runtime.
 *
 * @param {HTMLElement} container - Parent element containing .mermaid divs
 */
let mermaidReady = false
let currentMermaidTheme = null
let themeObserver = null

export async function initMermaidInElement(container) {
    if (!container) return
    const nodes = container.querySelectorAll('.mermaid:not([data-processed])')
    if (!nodes.length) return

    // Stash original source text before mermaid replaces it with SVG
    nodes.forEach(node => {
        if (!node.dataset.source) node.dataset.source = node.textContent
    })

    const { default: mermaid } = await import('mermaid')
    if (!mermaidReady) {
        currentMermaidTheme = getMermaidTheme()
        mermaid.initialize({
            startOnLoad: false,
            theme: currentMermaidTheme,
            securityLevel: 'strict',
        })
        mermaidReady = true
        installThemeObserver()
    }
    await mermaid.run({ nodes })
}

/**
 * Re-initialize mermaid and re-render ALL processed diagrams in the document
 * when the theme changes. Restores stashed source text from `data-source`,
 * resets `data-processed`, and calls `mermaid.run()` again.
 */
async function reRenderAllMermaid() {
    if (!mermaidReady) return
    const newTheme = getMermaidTheme()
    if (newTheme === currentMermaidTheme) return

    const { default: mermaid } = await import('mermaid')
    currentMermaidTheme = newTheme
    mermaid.initialize({
        startOnLoad: false,
        theme: currentMermaidTheme,
        securityLevel: 'strict',
    })

    // Reset all processed diagrams to their original source text
    const processed = document.querySelectorAll('.mermaid[data-processed]')
    processed.forEach(node => {
        if (node.dataset.source) {
            node.removeAttribute('data-processed')
            node.textContent = node.dataset.source
        }
    })

    if (processed.length) await mermaid.run({ nodes: processed })
}

/**
 * Watch for theme changes on <html data-theme="…"> and re-render diagrams.
 * Same MutationObserver pattern used by CodeEditor.vue.
 */
function installThemeObserver() {
    if (themeObserver) return
    themeObserver = new MutationObserver(() => reRenderAllMermaid())
    themeObserver.observe(document.documentElement, {
        attributes: true,
        attributeFilter: ['data-theme'],
    })
}
