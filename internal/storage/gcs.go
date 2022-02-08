
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
)

// Examples mostly culled from
//   https://github.com/GoogleCloudPlatform/golang-samples/tree/main/storage

type GoogleCloud struct {
	bucket string
}

func NewGoogleCloud(bucket string) GoogleCloud {
	return GoogleCloud{bucket}
}

func (s GoogleCloud) Name() string {
	return fmt.Sprintf("gcs::%s", s.bucket)
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

func (s GoogleCloud) WriteFile(file string, content []byte) error {
//	bucket := s.bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	// rc, err := client.Bucket(bucket).Object(file).NewReader(ctx)
	// if err != nil {
	// 	return nil, fmt.Errorf("Object(%q).NewReader: %v", file, err)
	// }
	// defer rc.Close()

	// data, err := ioutil.ReadAll(rc)
	// if err != nil {
	// 	return nil, fmt.Errorf("ioutil.ReadAll: %v", err)
	// }
	return nil
}

func (s GoogleCloud) DownloadFile(file string, outputFileName string) error {
	bucket := s.bucket
	ctx := context.Background()
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

	obj := client.Bucket(bucket).Object(file)
	_, err = obj.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("Object(%q).Attrs: %v", file, err)
	}
	
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
	// bucket := s.bucket
	// ctx := context.Background()
	// client, err := storage.NewClient(ctx)
	// if err != nil {
	// 	return fmt.Errorf("storage.NewClient: %v", err)
	// }
	// defer client.Close()
	
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()

	// f, err := os.Create(outputFileName)
	// if err != nil {
	// 	return fmt.Errorf("os.Create: %v", err)
	// }

	// obj := client.Bucket(bucket).Object(file)
	// _, err = obj.Attrs(ctx)
	// if err != nil {
	// 	return fmt.Errorf("Object(%q).Attrs: %v", file, err)
	// }
	
	// rc, err := obj.NewReader(ctx)
	// if err != nil {
	// 	return fmt.Errorf("Object(%q).NewReader: %v", file, err)
	// }
	// defer rc.Close()

	// if _, err := io.Copy(f, rc); err != nil {
	// 	return fmt.Errorf("io.Copy: %v", err)
	// }

	// if err = f.Close(); err != nil {
	// 	return fmt.Errorf("f.Close: %v", err)
	// }

	return nil
}
