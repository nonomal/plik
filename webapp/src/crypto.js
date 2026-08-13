import { Encrypter, Decrypter } from 'age-encryption'

/**
 * Generate a cryptographically secure random passphrase.
 * 32 alphanumeric characters using crypto.getRandomValues().
 * @returns {string}
 */
export function generatePassphrase() {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
    const array = new Uint32Array(32)
    crypto.getRandomValues(array)
    return Array.from(array, (v) => chars[v % chars.length]).join('')
}

/**
 * Encrypt a File/Blob using age passphrase encryption (streaming).
 * Returns an encrypted Blob suitable for upload.
 *
 * @param {File|Blob} file - The file to encrypt
 * @param {string} passphrase - The passphrase to encrypt with
 * @returns {Promise<Blob>} The encrypted data as a Blob
 */
export async function encryptFile(file, passphrase) {
    const encrypter = new Encrypter()
    encrypter.setPassphrase(passphrase)

    const stream = file.stream()
    const encryptedStream = await encrypter.encrypt(stream)

    // Convert the encrypted ReadableStream to a Blob
    return new Response(encryptedStream).blob()
}

/**
 * Decrypt an encrypted ReadableStream using age passphrase decryption.
 * Returns a decrypted Blob.
 *
 * @param {ReadableStream} stream - The encrypted stream
 * @param {string} passphrase - The passphrase to decrypt with
 * @returns {Promise<Blob>} The decrypted data as a Blob
 */
export async function decryptStream(stream, passphrase) {
    const decrypter = new Decrypter()
    decrypter.addPassphrase(passphrase)

    const decryptedStream = await decrypter.decrypt(stream)

    // Convert the decrypted ReadableStream to a Blob
    return new Response(decryptedStream).blob()
}

/**
 * Decrypt a fetched response body using age passphrase decryption.
 * Convenience wrapper that fetches the URL and decrypts the response.
 *
 * @param {string} url - The URL to fetch the encrypted file from
 * @param {string} passphrase - The passphrase to decrypt with
 * @returns {Promise<Blob>} The decrypted data as a Blob
 */
export async function fetchAndDecrypt(url, passphrase) {
    const response = await fetch(url)
    if (!response.ok) {
        throw new Error(`Failed to fetch: ${response.status} ${response.statusText}`)
    }
    return decryptStream(response.body, passphrase)
}
