// In-memory upload token store
// Tokens are never persisted to storage or URLs — they live only in the current tab's memory
const tokens = new Map()

export function setToken(uploadId, token) {
    if (token) tokens.set(uploadId, token)
}

export function getToken(uploadId) {
    return tokens.get(uploadId) || ''
}

export function clearToken(uploadId) {
    tokens.delete(uploadId)
}
