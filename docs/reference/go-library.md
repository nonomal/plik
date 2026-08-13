# Go Library

The `plik` package provides a public Go client library for the Plik API.

## Installation

```bash
go get -v github.com/root-gg/plik/plik
```

## Easy Mode

Quick one-liners for simple uploads:

```go
client := plik.NewClient("https://plik.server.url")

// Upload a file from path
upload, file, err := client.UploadFile("/home/file1")

// Upload from an io.Reader
upload, file, err := client.UploadReader("filename", ioReader)
```

## Full Mode

Complete workflow with all configuration options:

```go
client := plik.NewClient("https://plik.server.url")

// Optional client configuration
client.OneShot = true
client.Token = "xxxx-xxxx-xxxx-xxxx"

upload := client.NewUpload()

// Optional upload configuration
upload.OneShot = false

// Create file from path
file1, err := upload.AddFileFromPath(path)

// Create file from reader
file2, err := upload.AddFileFromReader("filename", ioReader)

// Create upload server side (optional step that is called by upload.Upload() / file.Upload() if omitted)
err = upload.Create()

// Upload all added files in parallel
err = upload.Upload()

// Upload a single file
err = file.Upload()

// Get upload URL
uploadURL, err := upload.GetURL()

// Get file URLs
for _, file := range upload.Files() {
    fileURL, err := file.GetURL()
}
```

## Advanced Operations

Download, delete, and manage existing uploads:

```go
// Get existing upload
upload := client.GetUpload(id)

// Download file
reader, err := upload.Files()[0].Download()

// Download as zip archive
reader, err := upload.DownloadZipArchive()

// Remove file (requires authentication)
err = upload.Files()[0].Delete()

// Remove upload (requires authentication)
err = upload.Delete()

// Add more files to existing upload (requires authentication)
err = upload.AddFileFromPath(path)
err = upload.Upload()

// Get remote server version
buildInfo, err := client.GetServerVersion()
```

## API Reference

### Client

| Method | Description |
|--------|-------------|
| `NewClient(url)` | Create new client with server URL |
| `client.NewUpload()` | Create new upload builder |
| `client.UploadFile(path)` | Quick upload from file path |
| `client.UploadReader(name, reader)` | Quick upload from io.Reader |
| `client.GetUpload(id)` | Retrieve existing upload |
| `client.Login` / `client.Password` | Set credentials |
| `client.Token` | Set upload token |

### Upload

| Method | Description |
|--------|-------------|
| `upload.AddFileFromPath(path)` | Add file from filesystem |
| `upload.AddFileFromReader(name, reader)` | Add file from io.Reader |
| `upload.Create()` | Create upload server-side |
| `upload.Upload()` | Upload all files |
| `upload.GetURL()` | Get upload URL |
| `upload.Delete()` | Remove upload |
| `upload.DownloadZipArchive()` | Download all files as zip |
| `upload.TTL` | Set TTL (seconds) |
| `upload.OneShot` / `upload.Stream` / `upload.Removable` | Upload options |

### File

| Method | Description |
|--------|-------------|
| `file.GetURL()` | Get download URL |
| `file.Download()` | Download file content |
| `file.Upload()` | Upload single file |
| `file.Delete()` | Remove file |
