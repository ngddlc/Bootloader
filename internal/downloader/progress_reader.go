package downloader

import (
	"io"
	"sync/atomic"
)

type progressReader struct {
	reader  io.Reader
	counter *atomic.Int64
}

func newProgressReader(r io.Reader, counter *atomic.Int64) *progressReader {
	return &progressReader{
		reader:  r,
		counter: counter,
	}
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.counter.Add(int64(n))
	}
	return n, err
}
