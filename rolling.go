package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
)

type rollingFile struct {
	mu sync.Mutex

	closed bool

	maxFileFrag int
	maxFragSize int64

	file     *os.File
	basePath string
	filePath string
	fileFrag int
	fragSize int64
}

var ErrClosedRollingFile = errors.New("rolling file is closed")

func (r *rollingFile) rollingName() error {
	var maxFileFrag = r.maxFileFrag - 1
	maxFilePath := fmt.Sprintf("%s.%d.log", r.basePath, maxFileFrag)
	err := os.Remove(maxFilePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for i := maxFileFrag - 1; i >= 0; i-- {
		oldFilePath := fmt.Sprintf("%s.%d.log", r.basePath, i)
		newFilePath := fmt.Sprintf("%s.%d.log", r.basePath, i+1)
		err := os.Rename(oldFilePath, newFilePath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func (r *rollingFile) roll() error {
	var rolling bool

	if r.file != nil {
		if r.fragSize < r.maxFragSize {
			return nil
		}
		r.file.Close()
		r.file = nil
		r.fragSize = 0
		rolling = true
	} else {
		fi, err := os.Stat(r.filePath)
		if err == nil {
			fileSize := fi.Size()
			if fileSize < r.maxFragSize {
				r.fragSize = fileSize
			} else {
				r.fragSize = 0
				rolling = true
			}
		}
	}

	if rolling {
		err := r.rollingName()
		if err != nil {
			return err
		}
	}

	f, err := os.OpenFile(r.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	} else {
		r.file = f
		return nil
	}
}

func (r *rollingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true
	if f := r.file; f != nil {
		r.file = nil
		return f.Close()
	}
	return nil
}

func (r *rollingFile) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0, ErrClosedRollingFile
	}

	if err := r.roll(); err != nil {
		return 0, err
	}

	n, err := r.file.Write(b)
	r.fragSize += int64(n)
	if err != nil {
		return n, err
	} else {
		return n, nil
	}
}

func NewRollingFile(basePath string, maxFileFrag int, maxFragSize int64) (io.WriteCloser, error) {
	if maxFileFrag <= 0 {
		return nil, fmt.Errorf("invalid max file-frag = %d", maxFileFrag)
	}
	if maxFragSize <= 0 {
		return nil, fmt.Errorf("invalid max frag-size = %d", maxFragSize)
	}

	dir, file := path.Split(basePath)
	if file == "" {
		return nil, fmt.Errorf("invalid base-path = %s, file name is required", basePath)
	}

	err := os.MkdirAll(dir, os.ModeDir|os.ModePerm)
	if err != nil {
		return nil, err
	}

	fileFrag := 0
	filePath := fmt.Sprintf("%s.%d.log", basePath, fileFrag)

	return &rollingFile{
		maxFileFrag: maxFileFrag,
		maxFragSize: maxFragSize,

		basePath: basePath,
		filePath: filePath,
		fileFrag: fileFrag,
	}, nil
}
