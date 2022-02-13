
package util

import (
    "io"
)

type NegateWriter struct {
	w io.Writer
}

// From
// https://stackoverflow.com/questions/64415363/calculate-crc32-checksum-from-file-reader-with-go-and-cloud-storage

func NewNegateWriter(w io.Writer) *NegateWriter {
	return &NegateWriter{
		w: w,
	}
	
}

func (neg *NegateWriter) Write(p []byte) (n int, err error) {
	// Negate everything
	for i := 0; i < len(p) ; i++ {
		p[i] = ^p[i]
	}
	n, err = neg.w.Write(p)
	return
}
