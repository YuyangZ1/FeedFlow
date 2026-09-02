package backend

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
)

// GCSClient is the interface the service layer depends on, so tests can
// substitute a mock instead of talking to a real GCS bucket.
type GCSClient interface {
	SaveToGCS(r io.Reader, objectName string) (string, error)
	DeleteFromGCS(objectName string) error
}

var (
	GCSBackend GCSClient
)

type GoogleCloudStorageBackend struct {
	client *storage.Client
	bucket string
}

type LocalStorageBackend struct {
	directory string
	baseURL   string
}

func InitGCSBackend() {
	if os.Getenv("STORAGE_BACKEND") == "local" {
		directory := getEnv("LOCAL_STORAGE_DIR", "uploads")
		if err := os.MkdirAll(directory, 0755); err != nil {
			panic(err)
		}
		GCSBackend = &LocalStorageBackend{
			directory: directory,
			baseURL:   getEnv("LOCAL_MEDIA_BASE_URL", "http://localhost:8080/media"),
		}
		fmt.Printf("Local media storage is enabled at %s\n", directory)
		return
	}

	client, err := storage.NewClient(context.Background())
	if err != nil {
		panic(err)
	}

	bucket := os.Getenv("GCS_BUCKET")
	if bucket == "" {
		panic("GCS_BUCKET environment variable must be set")
	}

	GCSBackend = &GoogleCloudStorageBackend{
		client: client,
		bucket: bucket,
	}
}

func (backend *GoogleCloudStorageBackend) SaveToGCS(r io.Reader, objectName string) (string, error) {
	ctx := context.Background()
	object := backend.client.Bucket(backend.bucket).Object(objectName)
	wc := object.NewWriter(ctx)
	if _, err := io.Copy(wc, r); err != nil {
		return "", err
	}

	if err := wc.Close(); err != nil {
		return "", err
	}

	if err := object.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return "", err
	}

	attrs, err := object.Attrs(ctx)
	if err != nil {
		return "", err
	}

	fmt.Printf("File is saved to GCS: %s\n", attrs.MediaLink)
	return attrs.MediaLink, nil
}

func (backend *GoogleCloudStorageBackend) DeleteFromGCS(objectName string) error {
	return backend.client.Bucket(backend.bucket).Object(objectName).Delete(context.Background())
}

func (backend *LocalStorageBackend) SaveToGCS(r io.Reader, objectName string) (string, error) {
	path := filepath.Join(backend.directory, filepath.Base(objectName))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s", backend.baseURL, filepath.Base(objectName))
	fmt.Printf("File is saved locally: %s\n", url)
	return url, nil
}

func (backend *LocalStorageBackend) DeleteFromGCS(objectName string) error {
	path := filepath.Join(backend.directory, filepath.Base(objectName))
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func LocalStorageDirectory() string {
	return getEnv("LOCAL_STORAGE_DIR", "uploads")
}
