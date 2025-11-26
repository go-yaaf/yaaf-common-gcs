// Package gcsfilestore provides a file system implementation for Google Cloud Storage.
// It allows to interact with GCS objects as if they were local files.
package gcsfilestore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"cloud.google.com/go/storage"

	fs "github.com/go-yaaf/yaaf-common/files"
)

// GcsFile is a concrete implementation of fs.IFile for Google Cloud Storage.
// It provides methods to read, write, and manage files in a GCS bucket.
//
// IMPORTANT: To use GCS from a local machine, you must first set up authentication by creating a service account
// and setting the GOOGLE_APPLICATION_CREDENTIALS environment variable.
// See https://cloud.google.com/storage/docs/reference/libraries#client-libraries-install-go for more details.
type GcsFile struct {
	uri      string
	path     string
	gsClient *storage.Client
	context  context.Context

	reader *storage.Reader
	writer *storage.Writer
}

// NewGcsFile creates a new GcsFile instance for the given GCS URI.
// The URI should be in the format "gs://<bucket-name>/<object-name>".
func NewGcsFile(uri string) fs.IFile {
	ctx := context.Background()
	return &GcsFile{uri: uri, context: ctx}
}

// URI returns the resource URI with the "gs" schema.
func (t *GcsFile) URI() string {
	return t.uri
}

// Close closes the underlying GCS client connection.
// It's important to call this method when you're done with the file to release resources.
func (t *GcsFile) Close() error {
	if t.gsClient != nil {
		return t.gsClient.Close()
	} else {
		return nil
	}
}

// Read implements the io.Reader interface. It reads data from the GCS object into the provided byte slice.
func (t *GcsFile) Read(p []byte) (int, error) {
	// ensure client
	if er := t.ensureClient(); er != nil {
		return 0, er
	}

	if t.reader != nil {
		return t.reader.Read(p)
	}

	// If reader is not open
	if bucket, object, err := parseUri(t.uri); err != nil {
		return 0, err
	} else {
		t.reader, err = t.gsClient.Bucket(bucket).Object(object).NewReader(t.context)
		if err != nil {
			return 0, err
		}
		return t.reader.Read(p)
	}
}

// Write implements the io.Writer interface. It writes data from the provided byte slice to the GCS object.
func (t *GcsFile) Write(p []byte) (int, error) {
	// ensure client
	if er := t.ensureClient(); er != nil {
		return 0, er
	}

	if t.writer != nil {
		return t.writer.Write(p)
	}

	// If reader is not open
	if bucket, object, err := parseUri(t.uri); err != nil {
		return 0, err
	} else {
		t.writer = t.gsClient.Bucket(bucket).Object(object).NewWriter(t.context)
		if t.writer == nil {
			return 0, fmt.Errorf("could not create writer")
		}
		return t.writer.Write(p)
	}
}

// ReadAll reads the entire content of the GCS object into a byte slice.
func (t *GcsFile) ReadAll() ([]byte, error) {
	// ensure client
	if err := t.ensureClient(); err != nil {
		return nil, err
	}

	// Get bucket and object from path
	bucket, object, err := parseUri(t.uri)
	if err != nil {
		return nil, err
	}

	rc, err := t.gsClient.Bucket(bucket).Object(object).NewReader(t.context)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// WriteAll writes the entire content of the byte slice to the GCS object.
func (t *GcsFile) WriteAll(b []byte) (int, error) {
	// ensure client
	if er := t.ensureClient(); er != nil {
		return 0, er
	}

	// Get bucket and object from path
	bucket, object, err := parseUri(t.uri)
	if err != nil {
		return 0, err
	}

	wc := t.gsClient.Bucket(bucket).Object(object).NewWriter(t.context)

	r := bytes.NewReader(b)
	if written, er := io.Copy(wc, r); er != nil {
		return int(written), er
	}
	if err := wc.Close(); err != nil {
		return 0, err
	}
	return 0, nil
}

// Exists checks if the GCS object exists.
func (t *GcsFile) Exists() bool {
	// ensure client
	if er := t.ensureClient(); er != nil {
		return false
	}

	// Get bucket and object from path
	bucket, object, err := parseUri(t.uri)
	if err != nil {
		return false
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return false
	}
	defer func() { _ = client.Close() }()

	_, err = client.Bucket(bucket).Object(object).Attrs(ctx)
	return err == nil
}

// Rename changes the name of the GCS object.
// The pattern can be a simple string or a template using {{path}}, {{file}}, and {{ext}}.
func (t *GcsFile) Rename(pattern string) (string, error) {
	// ensure client
	if er := t.ensureClient(); er != nil {
		return "", er
	}

	// Get bucket and object from path
	bucket, object, err := parseUri(t.uri)
	if err != nil {
		return "", err
	}

	// create new path based on pattern
	_, newPath, newFile, newExt, er := fs.ParseUri(pattern)
	if er != nil {
		return "", er
	}
	newUri := pattern
	newUri = strings.ReplaceAll(newUri, "{{path}}", newPath)
	newUri = strings.ReplaceAll(newUri, "{{file}}", newFile)
	newUri = strings.ReplaceAll(newUri, "{{ext}}", newExt)

	// Get bucket and object from path
	newBucket, newObject, err := parseUri(newUri)
	if err != nil {
		return "", err
	}

	src := t.gsClient.Bucket(bucket).Object(object)
	dst := t.gsClient.Bucket(newBucket).Object(newObject)

	if _, err = dst.CopierFrom(src).Run(t.context); err != nil {
		return "", fmt.Errorf("Object(%q).CopierFrom(%q).Run: %v", newObject, object, err)
	}
	if err = src.Delete(t.context); err != nil {
		return "", fmt.Errorf("Object(%q).Delete: %v", object, err)
	}

	t.uri = newUri
	return newUri, nil
}

// Delete deletes the GCS object.
func (t *GcsFile) Delete() error {
	// ensure client
	if er := t.ensureClient(); er != nil {
		return er
	}

	// Get bucket and object from path
	bucket, object, err := parseUri(t.uri)
	if err != nil {
		return err
	}

	return t.gsClient.Bucket(bucket).Object(object).Delete(t.context)
}

// Copy copies the content of the GCS object to the provided io.WriteCloser.
func (t *GcsFile) Copy(wc io.WriteCloser) (int64, error) {
	// ensure client
	if er := t.ensureClient(); er != nil {
		return 0, er
	}

	bucket, object, err := parseUri(t.uri)
	if err != nil {
		return 0, err
	}

	if reader, er := t.gsClient.Bucket(bucket).Object(object).NewReader(t.context); er != nil {
		return 0, er
	} else {
		written, cErr := io.Copy(wc, reader)
		_ = reader.Close()
		_ = wc.Close()
		return written, cErr
	}
}

// getReader returns an io.ReadCloser for the GCS object.
func (t *GcsFile) getReader() (io.ReadCloser, error) {
	if bucket, object, er := parseUri(t.uri); er != nil {
		return nil, er
	} else {
		return t.gsClient.Bucket(bucket).Object(object).NewReader(t.context)
	}
}

// getWriter returns an io.WriteCloser for the GCS object.
func (t *GcsFile) getWriter() (io.WriteCloser, error) {
	if bucket, object, er := parseUri(t.uri); er != nil {
		return nil, er
	} else {
		return t.gsClient.Bucket(bucket).Object(object).NewWriter(t.context), nil
	}
}

// ensureClient ensures that the GCS client is initialized.
func (t *GcsFile) ensureClient() error {
	if t.gsClient != nil {
		return nil
	}
	if cli, err := storage.NewClient(t.context); err != nil {
		return err
	} else {
		t.gsClient = cli
		return nil
	}
}

// parseUri parses a GCS URI and returns the bucket and object names.
func parseUri(uri string) (bucket string, object string, err error) {
	// Get bucket and object from path
	if Url, er := url.Parse(uri); er != nil {
		return "", "", er
	} else {
		return Url.Host, Url.Path[1:], nil
	}
}
