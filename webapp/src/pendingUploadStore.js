// In-memory store to pass initial files from UploadView → DownloadView across navigation
// Files + basicAuth + passphrase are stashed after createUpload(), consumed once by DownloadView on mount
const pending = new Map()

export function setPendingFiles(uploadId, files, basicAuth, passphrase = null, login = null, password = null) {
    pending.set(uploadId, { files, basicAuth, passphrase, login, password })
}

export function consumePendingFiles(uploadId) {
    const data = pending.get(uploadId)
    pending.delete(uploadId)
    return data || null
}
