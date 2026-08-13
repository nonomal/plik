import { rmSync, readFileSync } from 'fs'

/**
 * Global teardown — cleans up the temp directories created by start-server.sh
 * and start-server-subpath.sh.
 */
export default function globalTeardown() {
    for (const markerFile of ['/tmp/plik-e2e-tmpdir', '/tmp/plik-e2e-subpath-tmpdir']) {
        try {
            const dir = readFileSync(markerFile, 'utf-8').trim()
            if (dir.startsWith('/tmp/plik-e2e.') || dir.startsWith('/tmp/plik-e2e-subpath.')) {
                rmSync(dir, { recursive: true, force: true })
            }
            rmSync(markerFile, { force: true })
        } catch {
            // Already cleaned or never created — ignore
        }
    }
}

