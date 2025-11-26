// Package gcsfilestore provides a file system implementation for Google Cloud Storage.
// It allows to interact with GCS objects as if they were local files.
package gcsfilestore

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	fs "github.com/go-yaaf/yaaf-common/files"
)

// GcsFileStore is a concrete implementation of fs.IFileStore for Google Cloud Storage.
// It provides methods to list and manage files in a GCS bucket.
//
// IMPORTANT: To use GCS from a local machine, you must first set up authentication by creating a service account
// and setting the GOOGLE_APPLICATION_CREDENTIALS environment variable.
// See https://cloud.google.com/storage/docs/reference/libraries#client-libraries-install-go for more details.
type GcsFileStore struct {
	uri      string
	path     string
	gsClient *storage.Client
	context  context.Context
}

// NewGcsFileStore creates a new GcsFileStore instance for the given GCS URI.
// The URI should be in the format "gs://<bucket-name>/<object-prefix>".
func NewGcsFileStore(uri string) fs.IFileStore {
	ctx := context.Background()
	cli, _ := storage.NewClient(ctx)

	path := uri

	if Url, err := url.Parse(uri); err == nil {
		path = Url.Path
	}

	return &GcsFileStore{
		uri:      uri,
		path:     path,
		gsClient: cli,
		context:  ctx,
	}
}

// URI returns the resource URI with the "gs" schema.
func (f *GcsFileStore) URI() string {
	return f.uri
}

// List lists all files in the file store that match the given filter.
// The filter is a regular expression that is matched against the file's full URI.
func (f *GcsFileStore) List(filter string) ([]fs.IFile, error) {

	result := make([]fs.IFile, 0)
	cb := func(filePath string) {
		result = append(result, NewGcsFile(filePath))
	}

	err := f.Apply(filter, cb)
	return result, err
}

// Apply applies a given action to all files in the file store that match the given filter.
// The filter is a regular expression that is matched against the file's full URI.
// The action is a function that takes the file's URI as input.
func (f *GcsFileStore) Apply(filter string, action func(string)) error {

	// Get bucket and prefix from path
	bucket, prefix, err := parseUri(f.uri)
	if err != nil {
		return err
	}

	rgx, erx := regexp.Compile(filter)
	if erx != nil {
		if len(filter) > 0 {
			return erx
		}
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Minute*2)
	defer cancel()

	it := client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: fmt.Sprintf("%s/", prefix)})
	for {
		if attrs, er := it.Next(); er != nil {
			if er == iterator.Done {
				break
			} else {
				return er
			}
		} else {
			if attrs.Size > 0 {
				filePath := fmt.Sprintf("gcs://%s/%s", bucket, attrs.Name)
				if rgx == nil {
					action(filePath)
				} else {
					if rgx.MatchString(filePath) {
						action(filePath)
					}
				}
			}
		}
	}
	return nil
}

// Exists checks if a resource exists in the file store.
// The uri can be a full GCS URI or a relative path from the file store's root.
func (f *GcsFileStore) Exists(uri string) (result bool) {
	if strings.HasPrefix(uri, "gcs://") || strings.HasPrefix(uri, "gs://") {
		return NewGcsFile(uri).Exists()
	} else {
		return NewGcsFile(fs.CombineUri(f.uri, uri)).Exists()
	}
}

// Delete deletes a resource from the file store.
// The uri can be a full GCS URI or a relative path from the file store's root.
func (f *GcsFileStore) Delete(uri string) (err error) {
	if strings.HasPrefix(uri, "gcs://") || strings.HasPrefix(uri, "gs://") {
		return NewGcsFile(uri).Delete()
	} else {
		return NewGcsFile(fs.CombineUri(f.uri, uri)).Delete()
	}
}

// Close releases any associated resources, such as the GCS client.
func (f *GcsFileStore) Close() error {
	if f.gsClient != nil {
		return f.gsClient.Close()
	}
	return nil
}
