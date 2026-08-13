import { describe, it, expect } from 'vitest'
import { setPendingFiles, consumePendingFiles } from '../pendingUploadStore.js'

describe('pendingUploadStore', () => {
    it('returns null for non-existent key', () => {
        expect(consumePendingFiles('nonexistent')).toBeNull()
    })

    it('set + consume returns data', () => {
        const files = [{ fileName: 'test.txt', size: 100 }]
        const basicAuth = 'dXNlcjpwYXNz'
        setPendingFiles('upload1', files, basicAuth)

        const result = consumePendingFiles('upload1')
        expect(result).not.toBeNull()
        expect(result.files).toEqual(files)
        expect(result.basicAuth).toBe(basicAuth)
        expect(result.passphrase).toBeNull()
        expect(result.login).toBeNull()
        expect(result.password).toBeNull()
    })

    it('consumes once — second call returns null', () => {
        setPendingFiles('upload2', [{ fileName: 'a.txt' }], null)

        expect(consumePendingFiles('upload2')).not.toBeNull()
        expect(consumePendingFiles('upload2')).toBeNull()
    })

    it('stores passphrase when provided', () => {
        setPendingFiles('upload3', [], null, 'my-passphrase')

        const result = consumePendingFiles('upload3')
        expect(result.passphrase).toBe('my-passphrase')
    })

    it('stores different uploads independently', () => {
        setPendingFiles('a', [{ fileName: 'fileA' }], null)
        setPendingFiles('b', [{ fileName: 'fileB' }], null)

        const resultA = consumePendingFiles('a')
        const resultB = consumePendingFiles('b')
        expect(resultA.files[0].fileName).toBe('fileA')
        expect(resultB.files[0].fileName).toBe('fileB')
    })

    it('overwrite on re-set before consume', () => {
        setPendingFiles('upload4', [{ fileName: 'old.txt' }], null)
        setPendingFiles('upload4', [{ fileName: 'new.txt' }], 'auth2')

        const result = consumePendingFiles('upload4')
        expect(result.files[0].fileName).toBe('new.txt')
        expect(result.basicAuth).toBe('auth2')
    })

    it('stores login and password when provided', () => {
        setPendingFiles('upload5', [], null, null, 'myuser', 's3cret')

        const result = consumePendingFiles('upload5')
        expect(result.login).toBe('myuser')
        expect(result.password).toBe('s3cret')
    })
})
