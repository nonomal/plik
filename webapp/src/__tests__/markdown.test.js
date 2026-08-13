import { describe, it, expect } from 'vitest'
import { renderMarkdown } from '../markdown.js'

describe('renderMarkdown', () => {
    // ── Basic rendering ──

    it('returns empty string for falsy input', () => {
        expect(renderMarkdown(null)).toBe('')
        expect(renderMarkdown(undefined)).toBe('')
        expect(renderMarkdown('')).toBe('')
    })

    it('renders plain text as paragraph', () => {
        const html = renderMarkdown('hello world')
        expect(html).toContain('hello world')
        expect(html).toContain('<p>')
    })

    it('renders bold and italic', () => {
        expect(renderMarkdown('**bold**')).toContain('<strong>bold</strong>')
        expect(renderMarkdown('*italic*')).toContain('<em>italic</em>')
    })

    it('renders links', () => {
        const html = renderMarkdown('[plik](https://example.com)')
        expect(html).toContain('<a')
        expect(html).toContain('https://example.com')
    })

    it('renders line breaks (breaks: true)', () => {
        const html = renderMarkdown('line1\nline2')
        expect(html).toContain('<br')
    })

    it('renders code blocks', () => {
        const html = renderMarkdown('`inline code`')
        expect(html).toContain('<code>')
    })

    // ── Mermaid diagrams ──

    it('renders mermaid code blocks as <div class="mermaid">', () => {
        const md = '```mermaid\ngraph TD\n    A-->B\n```'
        const html = renderMarkdown(md)
        expect(html).toContain('<div class="mermaid">')
        expect(html).toContain('A--&gt;B')
        // Must NOT be wrapped in <pre><code>
        expect(html).not.toMatch(/<pre>.*mermaid/s)
    })

    it('leaves non-mermaid code blocks as <pre><code>', () => {
        const md = '```javascript\nconsole.log("hi")\n```'
        const html = renderMarkdown(md)
        expect(html).toContain('<pre>')
        expect(html).toContain('<code')
        expect(html).not.toContain('class="mermaid"')
    })

    it('mermaid content survives DOMPurify', () => {
        const md = '```mermaid\ngraph LR\n    A[Start] --> B{Check}\n```'
        const html = renderMarkdown(md)
        expect(html).toContain('<div class="mermaid">')
        expect(html).toContain('graph LR')
    })

    it('strips <script> tags inside mermaid node labels (XSS regression)', () => {
        const md = '```mermaid\ngraph TD\n    A[<script>alert("xss")</script>] --> B\n```'
        const html = renderMarkdown(md)
        expect(html).toContain('<div class="mermaid">')
        expect(html).not.toContain('<script')
    })

    // ── XSS sanitization ──

    it('strips <script> tags', () => {
        const html = renderMarkdown('<script>alert("xss")</script>')
        expect(html).not.toContain('<script')
        expect(html).not.toContain('alert')
    })

    it('strips onerror handlers on img tags', () => {
        const html = renderMarkdown('<img src=x onerror=alert(1)>')
        expect(html).not.toContain('onerror')
    })

    it('strips javascript: URLs', () => {
        const html = renderMarkdown('[click](javascript:alert(1))')
        expect(html).not.toContain('javascript:')
    })

    it('strips event handlers in HTML', () => {
        const html = renderMarkdown('<div onmouseover="alert(1)">hover</div>')
        expect(html).not.toContain('onmouseover')
    })

    it('strips iframe tags', () => {
        const html = renderMarkdown('<iframe src="https://evil.com"></iframe>')
        expect(html).not.toContain('<iframe')
    })
})
