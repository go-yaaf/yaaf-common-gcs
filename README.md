# GO-YAAF Google Cloud Storage (GCS) FileStore
[![Project status](https://img.shields.io/badge/version-1.0-green.svg)](https://github.com/go-yaaf/yaaf-common-gcs)
[![Build](https://github.com/go-yaaf/yaaf-common-gcs/actions/workflows/build.yml/badge.svg)](https://github.com/go-yaaf/yaaf-common-gcs/actions/workflows/build.yml)
[![Coverage Status](https://coveralls.io/repos/github/go-yaaf/yaaf-common-gcs/badge.svg?branch=main)](https://coveralls.io/github/go-yaaf/yaaf-common-gcs?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-yaaf/yaaf-common-gcs)](https://goreportcard.com/report/github.com/go-yaaf/yaaf-common-gcs)
[![GoDoc](https://godoc.org/github.com/go-yaaf/yaaf-common-gcs?status.svg)](https://pkg.go.dev/github.com/go-yaaf/yaaf-common-gcs)
[![License](https://img.shields.io/dub/l/vibe-d.svg)](https://github.com/go-yaaf/yaaf-common-gcs/blob/main/LICENSE)

This library provides a [Google Cloud Storage (GCS)](https://cloud.google.com/storage) based implementation of the `IFileStore` and `IFile` interfaces from the `yaaf-common` library. It allows you to interact with files in a GCS bucket as if they were part of a local file system.

## What It Does

This library abstracts the complexities of the Google Cloud Storage API behind a set of simple, familiar file system interfaces. You can:

- Read, write, and manage individual files in GCS.
- List files within a GCS bucket/prefix.
- Check for file existence.
- Delete files.
- Copy and rename files within and between buckets.

## Prerequisites

To use this library, you must first authenticate with Google Cloud. For local development, the recommended approach is to:

1.  Create a service account in your Google Cloud project.
2.  Download the JSON key file for the service account.
3.  Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the path of your JSON key file.

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/keyfile.json"
```

For more details, see the [Google Cloud client libraries documentation](https://cloud.google.com/storage/docs/reference/libraries#client-libraries-install-go).

## Installation

Use `go get` to install the library:

```bash
go get -u github.com/go-yaaf/yaaf-common-gcs
```

Then, import the `gcsfilestore` package into your code:

```go
import "github.com/go-yaaf/yaaf-common-gcs/gcsfilestore"
import "github.com/go-yaaf/yaaf-common/files"
```

## Code Examples

### Creating a File Store

A file store represents a bucket or a specific prefix (folder) within a bucket.

```go
// Create a file store for a specific bucket and prefix
store := gcsfilestore.NewGcsFileStore("gs://my-awesome-bucket/my-folder")
defer store.Close()
```

### Writing a File

You can get a file handle and write data to it. The file will be created if it doesn't exist.

```go
// Get a file handle
file, err := files.GetFile("gs://my-awesome-bucket/my-folder/hello.txt")
if err != nil {
    // Handle error
}
defer file.Close()

// Write content to the file
content := []byte("Hello, Google Cloud Storage!")
_, err = file.WriteAll(content)
if err != nil {
    // Handle error
}
```

### Reading a File

Reading a file is just as straightforward.

```go
// Get a file handle
file, err := files.GetFile("gs://my-awesome-bucket/my-folder/hello.txt")
if err != nil {
    // Handle error
}
defer file.Close()

// Read the entire file content
data, err := file.ReadAll()
if err != nil {
    // Handle error
}
fmt.Println(string(data)) // Output: Hello, Google Cloud Storage!
```

### Listing Files

You can list files within a file store's scope. The filter is a regular expression.

```go
store := gcsfilestore.NewGcsFileStore("gs://my-awesome-bucket/my-folder")
defer store.Close()

// List all .txt files in the store
fileList, err := store.List(".*\\.txt$")
if err != nil {
    // Handle error
}

for _, file := range fileList {
    fmt.Println("Found file:", file.URI())
}
```

### Deleting a File

Deleting a file is a simple operation.

```go
// Get a file handle
file, err := files.GetFile("gs://my-awesome-bucket/my-folder/hello.txt")
if err != nil {
    // Handle error
}
defer file.Close()

// Delete the file
if err = file.Delete(); err != nil {
    // Handle error
}
```

## Testing Locally

For local testing, it's highly recommended to use the [GCS emulator](https://github.com/fsouza/fake-gcs-server). You can run it using Docker:

```bash
docker run -d --name fake-gcs -p 4443:4443 fsouza/fake-gcs-server -scheme http
```

After starting the emulator, you must set the `STORAGE_EMULATOR_HOST` environment variable in your test environment:

```bash
export STORAGE_EMULATOR_HOST=localhost:4443
```

The library will automatically detect and use the emulator, disabling the need for real credentials during tests.
