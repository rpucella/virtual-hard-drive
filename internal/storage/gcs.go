
package storage

import (
	"context"
	"fmt"
	"time"
	"io"
	"io/ioutil"
	"os"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"rpucella.net/virtual-hard-drive/internal/util"
)

// Examples mostly culled from
//   https://github.com/GoogleCloudPlatform/golang-samples/tree/main/storage

const (
	UPLOAD_TIMEOUT = 120
)

type GoogleCloud struct {
	bucket string
}

func NewGoogleCloud(bucket string) GoogleCloud {
	return GoogleCloud{bucket}
}

func (s GoogleCloud) Name() string {
	return fmt.Sprintf("gcs::%s", s.bucket)
}

func log(msgs ...string) {
	acc := ""
	for _, msg := range msgs {
		acc += msg
	}
	fmt.Printf("[%s]\n", acc)
}

// Convert a UUID to a path on Cloud Storage.
// E.g.,
//   7b5d41cc-86d6-11eca8a3-0242ac120002
// to
//   7b/5d/41/cc/7b5d41cc-86d6-11eca8a3-0242ac120002

func (s GoogleCloud) UUIDToPath(uuid string) (string, error) {
	if len(uuid) != 36 {
		return "", fmt.Errorf("length of UUID %s <> 36", uuid)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s", uuid[:2], uuid[2:4], uuid[4:6], uuid[6:8], uuid), nil
}

func (s GoogleCloud) CatalogToPath(catalog string) (string, error) {
	return catalog, nil
}

func (s GoogleCloud) ListFiles() ([]string, error) {
	bucket := s.bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second * 10)
	defer cancel()

	var files []string
	it := client.Bucket(bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Bucket(%q).Objects: %v", bucket, err)
		}
		files = append(files, attrs.Name)
	}
	return files, nil
}

func ListBuckets(projectID string) ([]string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	var buckets []string
	it := client.Buckets(ctx, projectID)
	for {
		battrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, battrs.Name)
	}
	return buckets, nil
}

func (s GoogleCloud) ReadFile(file string) ([]byte, error) {
	bucket := s.bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	rc, err := client.Bucket(bucket).Object(file).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %v", file, err)
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll: %v", err)
	}
	return data, nil
}

func (s GoogleCloud) WriteFile(content []byte, target string) error {
	bucket := s.bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	wc := client.Bucket(bucket).Object(target).NewWriter(ctx)
	defer wc.Close()

	_, err = wc.Write(content)
	if err != nil {
		return fmt.Errorf("wc.Write: %v", err)
	}
	return nil
}

func (s GoogleCloud) DownloadFile(file string, outputFileName string) error {
	bucket := s.bucket
	ctx := context.Background()
	log("Connecting to ", bucket)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()
	
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	f, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("os.Create: %v", err)
	}

	log("Reading object ", file)
	obj := client.Bucket(bucket).Object(file)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("Object(%q).Attrs: %v", file, err)
	}
	log("Size = ", fmt.Sprintf("%d", attrs.Size))
	log("Starting download")
	rc, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("Object(%q).NewReader: %v", file, err)
	}
	defer rc.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}

	if err = f.Close(); err != nil {
		return fmt.Errorf("f.Close: %v", err)
	}
	return nil
}

func (s GoogleCloud) UploadFile(file string, target string) error {
	bucket := s.bucket
	ctx := context.Background()
	log("Connecting to ", bucket)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("os.Open: %v", err)
	}
	defer f.Close()

	// TODO: Split the file in chunks client-side and store in pieces?
	ctx, cancel := context.WithTimeout(ctx, time.Second * UPLOAD_TIMEOUT)
	defer cancel()

	log("Creating object ", target)
	obj := client.Bucket(bucket).Object(target)
	wc := obj.NewWriter(ctx)
	defer wc.Close()

	crcw := util.NewCRCwriter(wc)

	log("Starting upload")
	if _, err := io.Copy(crcw, f); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	// Wait a bit before closing, because... reasons?
	time.Sleep(5 * time.Second)
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}
	crc32c := crcw.Sum()
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("Object(%q).Attrs: %v", obj, err)
	}
	log("Checking CRC32C = ", fmt.Sprintf("%x", crc32c))
	if (crc32c != attrs.CRC32C) {
		return fmt.Errorf("crc32c of uploaded file different from %x", crc32c)
	}
	return nil
}

func (s GoogleCloud) RemoteInfo(target string) error {
	bucket := s.bucket
	ctx := context.Background()
	log("Connecting to ", bucket)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second * UPLOAD_TIMEOUT)
	defer cancel()

	log("Reading object ", target)
	attrs, err := client.Bucket(bucket).Object(target).Attrs(ctx)
	if err != nil {
		return fmt.Errorf("ObjectHandle.Attrs: %v", err)
	}
	fmt.Println()
	fmt.Printf("Bucket:       %s\n", attrs.Bucket)
	fmt.Printf("Name:         %s\n", attrs.Name)
	fmt.Printf("ContentType:  %s\n", attrs.ContentType)
	fmt.Printf("Size:         %d\n", attrs.Size)
	fmt.Printf("MD5:          %x\n", attrs.MD5)
	fmt.Printf("CRC32C:       %x\n", attrs.CRC32C)
	return nil
}
