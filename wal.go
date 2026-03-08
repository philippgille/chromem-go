package chromem

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WAL struct {
	dir string

	currentFile *os.File
	writer      *bufio.Writer

	segmentID      int
	currentSize    int64
	maxSegmentSize int64

	// mu guards all mutable WAL state (active file/writer, size, closed flag).
	// It also serializes writes and segment rotation.
	mu sync.Mutex

	// syncStopCh/syncWG manage the lifecycle of the background sync goroutine.
	syncStopCh chan struct{}
	syncWG     sync.WaitGroup

	// closeOnce makes Close idempotent.
	closeOnce sync.Once
	closed    bool
}

func NewWAL(path string, wal bool) (*WAL, error) {

	if !wal {
		return nil, nil
	}

	if !folderExists(path) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("couldn't create WAL directory: %w", err)
		}
	}

	walObj := &WAL{
		dir:            path,
		maxSegmentSize: 2 * 1024 * 1024,
		segmentID:      1,
		syncStopCh:     make(chan struct{}),
	}

	if err := walObj.createSegment(); err != nil {
		return nil, err
	}

	walObj.startSyncLoop()

	return walObj, nil
}

func folderExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && info.IsDir()
}

func (w *WAL) Append(obj any, compress bool, encryptionKey string) error {
	// WAL writes are routed through WriteBinary, which locks and writes to the
	// current active segment writer. We intentionally pass nil here to avoid
	// capturing a stale writer pointer during segment rotation.
	if err := persistToWriter(nil, obj, compress, w, encryptionKey); err != nil {
		return err
	}

	return w.rotateIfNeeded()
}

func (w *WAL) startSyncLoop() {
	// Periodically flush and fsync to bound data loss on process crash.
	ticker := time.NewTicker(1 * time.Second)

	w.syncWG.Add(1)
	go func() {
		defer w.syncWG.Done()
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.mu.Lock()
				if !w.closed {
					_ = w.writer.Flush()
					_ = w.currentFile.Sync()
				}
				w.mu.Unlock()
			case <-w.syncStopCh:
				return
			}
		}
	}()
}

func (w *WAL) Close() error {
	if w == nil {
		return nil
	}

	var closeErr error
	w.closeOnce.Do(func() {
		close(w.syncStopCh)
		w.syncWG.Wait()

		w.mu.Lock()
		defer w.mu.Unlock()

		if w.closed {
			return
		}

		if w.writer != nil {
			if err := w.writer.Flush(); err != nil {
				closeErr = errors.Join(closeErr, fmt.Errorf("couldn't flush WAL writer: %w", err))
			}
		}
		if w.currentFile != nil {
			if err := w.currentFile.Sync(); err != nil {
				closeErr = errors.Join(closeErr, fmt.Errorf("couldn't sync WAL file: %w", err))
			}
			if err := w.currentFile.Close(); err != nil {
				closeErr = errors.Join(closeErr, fmt.Errorf("couldn't close WAL file: %w", err))
			}
		}

		w.writer = nil
		w.currentFile = nil
		w.closed = true
	})

	return closeErr
}

func (w *WAL) replayWAL(walPath string, c *Collection) error {
	f, err := os.Open(walPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for {
		data, err := ReadBinary(f)

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("couldn't read WAL record: %w", err)
		}

		// decode document using existing reader logic
		d := &Document{}
		r := bytes.NewReader(data)

		err = readFromReader(r, d, "")
		if err != nil {
			return fmt.Errorf("couldn't decode WAL record: %w", err)
		}

		// apply document
		c.documentsLock.Lock()
		c.documents[d.ID] = d
		c.documentsLock.Unlock()
	}

	return nil
}

func (w *WAL) createSegment() error {

	filename := fmt.Sprintf("%06d.wal", w.segmentID)

	path := filepath.Join(w.dir, filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w.currentFile = f
	w.writer = bufio.NewWriter(f)

	w.currentSize = 0

	return nil
}

func (w *WAL) rotateIfNeeded() error {

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("WAL is closed")
	}

	if w.currentSize < w.maxSegmentSize {
		return nil
	}

	// Rotate only while holding the same lock used by WriteBinary so no write can
	// target a file descriptor that is being closed.
	w.writer.Flush()
	w.currentFile.Close()

	w.segmentID++

	return w.createSegment()
}
