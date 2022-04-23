
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
	"path"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"rpucella.net/virtual-hard-drive/internal/util"
)

// Examples mostly culled from
//   https://github.com/GoogleCloudPlatform/golang-samples/tree/main/storage

const (
	UPLOAD_TIMEOUT = 600    // 10m
	DOWNLOAD_TIMEOUT = 600  // 10m
	// CHUNK_SIZE = 52428800   // 50MB
	CHUNK_SIZE = 52428800 * 4  // 200MB
	UPLOAD_PAUSE = 2
)

const CONFIG_FOLDER = ".vhd"
const CONFIG_CREDENTIALS = "priv.json"

func keyFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil { 
		return "", fmt.Errorf("cannot get home directory: %v", err)
	}
	configFolder := path.Join(home, CONFIG_FOLDER)
	info, err := os.Stat(configFolder)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("config folder %s does not exist", configFolder)
	} else if err != nil {
		return "", fmt.Errorf("cannot access %s directory: %w", configFolder, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("path %s not a directory", configFolder)
	}
	return path.Join(configFolder, CONFIG_CREDENTIALS), nil
}


type GoogleCloud struct {
	bucket string
	privKey string
}

func NewGoogleCloud(bucket string) (GoogleCloud, error) {
	privKey, err := keyFile()
	if err != nil {
		return GoogleCloud{}, err
	}
	return GoogleCloud{bucket, privKey}, nil
}

func (s GoogleCloud) Name() string {
	return fmt.Sprintf("gcs::%s", s.bucket)
}

// Convert a UUID to a path on Cloud Storage.
// E.g.,
//   7b5d41cc-86d6-11eca8a3-0242ac120002
// to
//   7b/5d/41/cc/7b5d41cc-86d6-11eca8a3-0242ac120002

func uuidToPath(uuid string) (string, error) {
	if len(uuid) != 36 {
		return "", fmt.Errorf("length of UUID %s <> 36", uuid)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s", uuid[:2], uuid[2:4], uuid[4:6], uuid[6:8], uuid), nil
}

func (s GoogleCloud) log(text string) {
	fmt.Printf("[gcs] %s\n", text)
}

func (s GoogleCloud) ListFiles() ([]string, error) {
	bucket := s.bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(s.privKey))
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

func (s GoogleCloud) ReadFile(file string) ([]byte, error) {
	bucket := s.bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(s.privKey))
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
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(s.privKey))
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
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(s.privKey))
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()
	
	target, err := uuidToPath(uuid)
	if err != nil {
		return err
	}
	
	f, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("os.Create: %v", err)
	}
	negf := util.NewNegateWriter(f)

	if metadata != "" {
		// We have chunks.
		numParts, err := strconv.ParseInt(metadata, 10, 64)
		if err != nil {
			return fmt.Errorf("wrong metadata: %s", metadata)
		}
		s.log(fmt.Sprintf("objects: %d", numParts))
		for i := int64(0); i < numParts; i++ {
			currTarget := fmt.Sprintf("%s.%03d", target, i)
			s.log(fmt.Sprintf("downloading object %s", currTarget))

			// Setup a timeout.
			ctx, cancel := context.WithTimeout(ctx, time.Second * DOWNLOAD_TIMEOUT)
			defer cancel()
			
			obj := client.Bucket(bucket).Object(currTarget)
			rc, err := obj.NewReader(ctx)
			if err != nil {
				return fmt.Errorf("Object(%q).NewReader: %w", currTarget, err)
			}
			defer rc.Close()
			
			if _, err := io.Copy(negf, rc); err != nil {
				return fmt.Errorf("io.Copy: %w", err)
			}
		
			if err = rc.Close(); err != nil {
				return fmt.Errorf("rc.Close: %w", err)
			}
		}
	} else {
		s.log(fmt.Sprintf("downloading object %s", target))

		// Setup a timeout.
		ctx, cancel := context.WithTimeout(ctx, time.Second * DOWNLOAD_TIMEOUT)
		defer cancel()
		
		obj := client.Bucket(bucket).Object(target)
		rc, err := obj.NewReader(ctx)
		if err != nil {
			return fmt.Errorf("Object(%q).NewReader: %v", target, err)
		}
		defer rc.Close()
		
		if _, err := io.Copy(negf, rc); err != nil {
			return fmt.Errorf("io.Copy: %v", err)
		}
	}
	
	if err = f.Close(); err != nil {
		return fmt.Errorf("f.Close: %w", err)
	}
	
	return nil
}

func (s GoogleCloud) UploadFile(path string, uuid string) (string, error) {
	bucket := s.bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(s.privKey))
	if err != nil {
		return "", fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	target, err := uuidToPath(uuid)
	if err != nil {
		return "", err
	}

	// Get source file size.
	attrs, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("os.Stat: %v", err)
	}
	fileSize := attrs.Size()
	// Calculate total number of parts the file will be chunked into.
	totalPartsNum := int(math.Ceil(float64(fileSize) / float64(CHUNK_SIZE)))
	s.log(fmt.Sprintf("objects: %d", totalPartsNum))

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("os.Open: %v", err)
	}
	defer f.Close()

	// From:
	// https://socketloop.com/tutorials/golang-how-to-split-or-chunking-a-file-to-smaller-pieces
	for i := 0; i < totalPartsNum; i++ {
		currTarget := fmt.Sprintf("%s.%03d", target, i)
		s.log(fmt.Sprintf("uploading object %s", currTarget))

		// Setup a timeout.
		ctx, cancel := context.WithTimeout(ctx, time.Second * UPLOAD_TIMEOUT)
		defer cancel()
		
		obj := client.Bucket(bucket).Object(currTarget)
		wc := obj.NewWriter(ctx)
		defer wc.Close()
		// Order is important: first we flip bytes, then we compute the CRC.
		negw := util.NewNegateWriter(wc)
		crcw := util.NewCRCWriter(negw)
		
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
		if (crc32c != attrs.CRC32C) {
			return "", fmt.Errorf("crc32c of uploaded file different from %x", crc32c)
		}

		// Cancel the timeout.
		cancel()
	}
	metadata := fmt.Sprintf("%d", totalPartsNum)
	return metadata, nil
}

func (s GoogleCloud) uploadFileSingle_OBSOLETE(path string, target string) error {
	bucket := s.bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(s.privKey))
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

	fmt.Printf("Uploading to object %s\n", target)
	obj := client.Bucket(bucket).Object(target)
	wc := obj.NewWriter(ctx)
	defer wc.Close()

	negw := util.NewNegateWriter(wc)
	crcw := util.NewCRCWriter(negw)

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
	if (crc32c != attrs.CRC32C) {
		return fmt.Errorf("crc32c of uploaded file different from %x", crc32c)
	}
	return nil
}

func (s GoogleCloud) RemoteInfo(uuid string, metadata string) error {
	bucket := s.bucket
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(s.privKey))
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
 	defer client.Close()

	target, err := uuidToPath(uuid)
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(ctx, time.Second * UPLOAD_TIMEOUT)
	defer cancel()

	if metadata != "" {
		// We have chunks.
		numParts, err := strconv.ParseInt(metadata, 10, 64)
		if err != nil {
			return fmt.Errorf("wrong metadata: %s", metadata)
		}
		fmt.Printf("Remote:      %s\n", s.Name())
		for i := int64(0); i < numParts; i++ {
			currTarget := fmt.Sprintf("%s.%03d", target, i)
			attrs, err := client.Bucket(bucket).Object(currTarget).Attrs(ctx)
			if err != nil {
				return fmt.Errorf("ObjectHandle.Attrs: %v", err)
			}
			if attrs.Size < 1024 {
				fmt.Printf(" %s  %4d B  %x\n", attrs.Name, attrs.Size, attrs.CRC32C)
			} else if attrs.Size < 1024 * 1024 {
				size := attrs.Size / 1024
				fmt.Printf(" %s  %4d MiB  %x\n", attrs.Name, size, attrs.CRC32C)
			} else {
				size := attrs.Size / (1024 * 1024)
				fmt.Printf(" %s  %4d GiB  %x\n", attrs.Name, size, attrs.CRC32C)
			}
		}
	} else {
		attrs, err := client.Bucket(bucket).Object(target).Attrs(ctx)
		if err != nil {
			return fmt.Errorf("ObjectHandle.Attrs: %v", err)
		}
		fmt.Printf("Remote:      %s\n", s.Name())
		if attrs.Size < 1024 {
			fmt.Printf(" %s  %4d B  %x\n", attrs.Name, attrs.Size, attrs.CRC32C)
		} else if attrs.Size < 1024 * 1024 {
			size := attrs.Size / 1024
			fmt.Printf(" %s  %4d MiB  %x\n", attrs.Name, size, attrs.CRC32C)
		} else {
			size := attrs.Size / (1024 * 1024)
			fmt.Printf(" %s  %4d GiB  %x\n", attrs.Name, size, attrs.CRC32C)
		}
	}
	return nil
}
