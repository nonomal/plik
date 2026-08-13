import { describe, it, expect } from 'vitest'
import { languages } from '@codemirror/language-data'

// ── getLanguageFromFilename (replicated from CodeEditor.vue) ──
// This is a private function inside CodeEditor.vue, so we replicate the logic
// here against the same @codemirror/language-data to test extension matching.


function getLanguageFromFilename(filename) {
    if (!filename) return null
    const basename = filename.split('/').pop() || filename

    for (const lang of languages) {
        if (lang.filename && lang.filename.test(basename)) {
            return lang
        }
    }

    const ext = basename.split('.').pop()?.toLowerCase()
    if (!ext || ext === basename.toLowerCase()) return null

    for (const lang of languages) {
        if (lang.extensions && lang.extensions.includes(ext)) {
            return lang
        }
        if (lang.alias && lang.alias.includes(ext)) {
            return lang
        }
    }
    return null
}

// ── Extension matching ──

describe('getLanguageFromFilename', () => {
    it('detects Perl from .pl extension', () => {
        const lang = getLanguageFromFilename('script.pl')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Perl')
    })

    it('detects Perl from .pm extension', () => {
        const lang = getLanguageFromFilename('Module.pm')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Perl')
    })

    it('detects Python from .py extension', () => {
        const lang = getLanguageFromFilename('main.py')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Python')
    })

    it('detects JavaScript from .js extension', () => {
        const lang = getLanguageFromFilename('app.js')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('JavaScript')
    })

    it('detects JSON from .json extension', () => {
        const lang = getLanguageFromFilename('package.json')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('JSON')
    })

    it('detects Go from .go extension', () => {
        const lang = getLanguageFromFilename('main.go')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Go')
    })

    it('detects Rust from .rs extension', () => {
        const lang = getLanguageFromFilename('lib.rs')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Rust')
    })

    it('detects Shell from .sh extension', () => {
        const lang = getLanguageFromFilename('deploy.sh')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Shell')
    })

    it('detects YAML from .yaml extension', () => {
        const lang = getLanguageFromFilename('config.yaml')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('YAML')
    })

    it('detects TOML from .toml extension', () => {
        const lang = getLanguageFromFilename('Cargo.toml')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('TOML')
    })

    it('is case-insensitive on extension', () => {
        const lang = getLanguageFromFilename('SCRIPT.PL')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Perl')
    })

    it('returns null for unknown extension', () => {
        expect(getLanguageFromFilename('data.xyz')).toBeNull()
    })

    it('returns null for unrecognized extensionless filename', () => {
        expect(getLanguageFromFilename('CREDITS')).toBeNull()
    })

    it('returns null for empty filename', () => {
        expect(getLanguageFromFilename('')).toBeNull()
    })

    it('returns null for null filename', () => {
        expect(getLanguageFromFilename(null)).toBeNull()
    })


    it('detects Dockerfile via CM filename pattern', () => {
        const lang = getLanguageFromFilename('Dockerfile')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Dockerfile')
    })

    it('detects Gemfile via CM filename pattern', () => {
        const lang = getLanguageFromFilename('Gemfile')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Ruby')
    })

    it('detects Jenkinsfile via CM filename pattern', () => {
        const lang = getLanguageFromFilename('Jenkinsfile')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('Groovy')
    })

    it('handles dotfiles (no actual extension)', () => {
        // ".gitignore" → ext = "gitignore", no match expected
        expect(getLanguageFromFilename('.gitignore')).toBeNull()
    })

    it('uses last extension for multi-dot filenames', () => {
        const lang = getLanguageFromFilename('archive.test.js')
        expect(lang).not.toBeNull()
        expect(lang.name).toBe('JavaScript')
    })
})
