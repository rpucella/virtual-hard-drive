
package util

import (
    "hash"
    "hash/crc32"
    "io"
)

const (
	GCS_POLY = crc32.Castagnoli
)

// From
// https://stackoverflow.com/questions/64415363/calculate-crc32-checksum-from-file-reader-with-go-and-cloud-storage

func NewCRCWriter(w io.Writer) *CRCWriter {

	// Specific for this polynomial.
	return &CRCWriter{
		h: crc32.New(crc32.MakeTable(GCS_POLY)),
		w: w,
	}
	
}

type CRCWriter struct {
	h hash.Hash32
	w io.Writer
}

func (c *CRCWriter) Write(p []byte) (n int, err error) {
	n, err = c.w.Write(p)  // with each write ...
	c.h.Write(p)           // ... update the hash
	return
}

func (c *CRCWriter) Sum() uint32 {
	return c.h.Sum32() // final hash
}

