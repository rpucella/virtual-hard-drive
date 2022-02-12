
package storage

import (
	"context"
	"fmt"
	"time"
	"io"
	"io/ioutil"
	"os"
	"math"
	"strconv"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"rpucella.net/virtual-hard-drive/internal/util"
)

// Examples mostly culled from
//   https://github.com/GoogleCloudPlatform/golang-samples/tree/main/storage

const (
	UPLOAD_TIMEOUT = 120
	CHUNK_SIZE = 52428800   // 50MB
	UPLOAD_PAUSE = 2
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

func (s GoogleCloud) DownloadFile(uuid string, metadata string, outputFileName string) error {
	bucket := s.bucket
	ctx := context.Background()
	log("Connecting to ", bucket)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()
	
	target, err := s.UUIDToPath(uuid)
	if err != nil {
		return fmt.Errorf("RemoteInfo: %w", err)
	}
	
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	f, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("os.Create: %v", err)
	}

	if metadata != "" {
		// We have chunks.
		numParts, err := strconv.ParseInt(metadata, 10, 64)
		if err != nil {
			return fmt.Errorf("wrong metadata: %s", metadata)
		}
		for i := int64(0); i < numParts; i++ {
			currTarget := fmt.Sprintf("%s.%03d", target, i)
			log("Starting download ", currTarget)
			obj := client.Bucket(bucket).Object(currTarget)
			rc, err := obj.NewReader(ctx)
			if err != nil {
				return fmt.Errorf("Object(%q).NewReader: %w", currTarget, err)
			}
			defer rc.Close()
			
			if _, err := io.Copy(f, rc); err != nil {
				return fmt.Errorf("io.Copy: %w", err)
			}
		
			if err = rc.Close(); err != nil {
				return fmt.Errorf("rc.Close: %w", err)
			}
		}
	} else {
		log("Reading object ", target)
		obj := client.Bucket(bucket).Object(target)
		attrs, err := obj.Attrs(ctx)
		if err != nil {
			return fmt.Errorf("Object(%q).Attrs: %v", target, err)
		}
		log("Size = ", fmt.Sprintf("%d", attrs.Size))
		log("Starting download")
		rc, err := obj.NewReader(ctx)
		if err != nil {
			return fmt.Errorf("Object(%q).NewReader: %v", target, err)
		}
		defer rc.Close()
		
		if _, err := io.Copy(f, rc); err != nil {
			return fmt.Errorf("io.Copy: %v", err)
		}
	}
	
	if err = f.Close(); err != nil {
		return fmt.Errorf("f.Close: %w", err)
	}
	
	return nil
}

func (s GoogleCloud) UploadFile(path string, target string) (string, error) {
	bucket := s.bucket
	ctx := context.Background()
	log("Connecting to ", bucket)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	// Get source file size.
	attrs, err := os.Stat(path)
	if err != nil {
		fmt.Errorf("os.Stat: %v", err)
	}
	fileSize := attrs.Size()
	// Calculate total number of parts the file will be chunked into.
	totalPartsNum := int(math.Ceil(float64(fileSize) / float64(CHUNK_SIZE)))
	log("Calculating ", fmt.Sprintf("%d", totalPartsNum), " parts")

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("os.Open: %v", err)
	}
	defer f.Close()

	// TODO: Split the file in chunks client-side and store in pieces?
	ctx, cancel := context.WithTimeout(ctx, time.Second * UPLOAD_TIMEOUT)
	defer cancel()

	// From:
	// https://socketloop.com/tutorials/golang-how-to-split-or-chunking-a-file-to-smaller-pieces
	for i := 0; i < totalPartsNum; i++ {
		currTarget := fmt.Sprintf("%s.%03d", target, i)
		log("Creating object ", currTarget)
		obj := client.Bucket(bucket).Object(currTarget)
		wc := obj.NewWriter(ctx)
		defer wc.Close()
		crcw := util.NewCRCwriter(wc)
		log("Starting upload")
		
		partSize := int(math.Min(CHUNK_SIZE, float64(fileSize - int64(i * CHUNK_SIZE))))
		partBuffer := make([]byte, partSize)
		
		f.Read(partBuffer)
		
		// write to disk
		n, err := crcw.Write(partBuffer)
		if err != nil {
			return "", fmt.Errorf("Writer.Writer: %v", err)
		}
		if n < partSize {
			return "", fmt.Errorf("Too few bytes written - expected %d wrote %d", partSize, n)
		}
		time.Sleep(UPLOAD_PAUSE * time.Second)
		if err := wc.Close(); err != nil {
			return "", fmt.Errorf("Writer.Close: %v", err)
		}
		crc32c := crcw.Sum()
		attrs, err := obj.Attrs(ctx)
		if err != nil {
			return "", fmt.Errorf("Object(%q).Attrs: %v", obj, err)
		}
		log("Checking CRC32C = ", fmt.Sprintf("%x", crc32c))
		if (crc32c != attrs.CRC32C) {
			return "", fmt.Errorf("crc32c of uploaded file different from %x", crc32c)
		}
	}
	metadata := fmt.Sprintf("%d", totalPartsNum)
	return metadata, nil
}

func (s GoogleCloud) uploadFileSingle(path string, target string) error {
	bucket := s.bucket
	ctx := context.Background()
	log("Connecting to ", bucket)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	f, err := os.Open(path)
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

func (s GoogleCloud) RemoteInfo(uuid string, metadata string) error {
	bucket := s.bucket
	ctx := context.Background()
	log("Connecting to ", bucket)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
 	defer client.Close()

	target, err := s.UUIDToPath(uuid)
	if err != nil {
		return fmt.Errorf("RemoteInfo: %w", err)
	}
	
	ctx, cancel := context.WithTimeout(ctx, time.Second * UPLOAD_TIMEOUT)
	defer cancel()

	if metadata != "" {
		// We have chunks.
		numParts, err := strconv.ParseInt(metadata, 10, 64)
		if err != nil {
			return fmt.Errorf("wrong metadata: %s", metadata)
		}
		for i := int64(0); i < numParts; i++ {
			currTarget := fmt.Sprintf("%s.%03d", target, i)
			attrs, err := client.Bucket(bucket).Object(currTarget).Attrs(ctx)
			if err != nil {
				return fmt.Errorf("ObjectHandle.Attrs: %v", err)
			}
			fmt.Println()
			fmt.Printf("Bucket:       %s\n", attrs.Bucket)
			fmt.Printf("Name:         %s\n", attrs.Name)
			fmt.Printf("Size:         %d\n", attrs.Size)
			fmt.Printf("MD5:          %x\n", attrs.MD5)
			fmt.Printf("CRC32C:       %x\n", attrs.CRC32C)
		}
	} else {
		attrs, err := client.Bucket(bucket).Object(target).Attrs(ctx)
		if err != nil {
			return fmt.Errorf("ObjectHandle.Attrs: %v", err)
		}
		fmt.Println()
		fmt.Printf("Bucket:       %s\n", attrs.Bucket)
		fmt.Printf("Name:         %s\n", attrs.Name)
		fmt.Printf("Size:         %d\n", attrs.Size)
		fmt.Printf("MD5:          %x\n", attrs.MD5)
		fmt.Printf("CRC32C:       %x\n", attrs.CRC32C)
	}
	return nil
}
