package chromem

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WAL struct {
	path   string
	file   *os.File
	writer *bufio.Writer
	mu     sync.RWMutex
}

func NewWAL(path string, wal bool) (*WAL, error) {

	if !wal {
		return nil, nil
	}

	walFilePath := filepath.Join(path, "wl01.db")

	fi, err := os.Stat(walFilePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("couldn't get info about the path: %w", err)
		} else {
			// If the file doesn't exist, create the parent path
			err := os.MkdirAll(filepath.Dir(walFilePath), 0o700)
			if err != nil {
				return nil, fmt.Errorf("couldn't create parent directories to path: %w", err)
			}
		}
	} else if fi.IsDir() {
		return nil, fmt.Errorf("path is a directory: %s", walFilePath)
	}

	f, err := os.OpenFile(walFilePath, os.O_CREATE|os.O_RDWR, 0644)

	if err != nil {
		return nil, err
	}

	walObj := &WAL{
		path:   walFilePath,
		file:   f,
		writer: bufio.NewWriter(f),
	}

	walObj.startSyncLoop()

	return walObj, err
}

func (w *WAL) Append(obj any, compress bool, encryptionKey string) error {
	return persistToWriter(w.writer, obj, compress, w, encryptionKey)
}

func (w *WAL) startSyncLoop() {
	ticker := time.NewTicker(1 * time.Second)

	go func() {
		for range ticker.C {
			w.mu.Lock()
			w.writer.Flush()
			w.file.Sync()
			w.mu.Unlock()
		}
	}()
}
