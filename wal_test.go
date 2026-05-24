package chromem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestNewWAL_Disabled(t *testing.T) {
	wal, err := NewWAL(t.TempDir(), false, 0)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if wal != nil {
		t.Fatal("expected nil WAL when disabled, got", wal)
	}
}

func TestWAL_AppendReplay_CloseBlocksWrites(t *testing.T) {
	dir := t.TempDir()

	wal, err := NewWAL(dir, true, 0)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	doc := Document{
		ID:        "doc-1",
		Embedding: []float32{1, 0, 0},
		Content:   "hello wal",
	}

	err = wal.Append(doc, true, "")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	err = wal.Close()
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Close must be idempotent.
	err = wal.Close()
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	err = wal.Append(doc, true, "")
	if err == nil {
		t.Fatal("expected error when appending to closed WAL, got nil")
	}
	if !strings.Contains(err.Error(), "WAL is closed") {
		t.Fatal("expected closed WAL error, got", err)
	}

	collection := &Collection{
		documents: make(map[string]*Document),
	}

	walFiles, err := listWALSegments(dir)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if len(walFiles) != 1 {
		t.Fatal("expected 1 WAL file, got", len(walFiles))
	}

	err = wal.replayWAL(walFiles[0], collection)
	if err != nil {
		t.Fatal("expected no error replaying WAL, got", err)
	}

	if collection.Count() != 1 {
		t.Fatal("expected 1 replayed document, got", collection.Count())
	}
	replayed, err := collection.GetByID(context.Background(), doc.ID)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if replayed.Content != doc.Content {
		t.Fatal("expected content", doc.Content, "got", replayed.Content)
	}
}

func TestWAL_Rotate_ConcurrentAppend(t *testing.T) {
	dir := t.TempDir()

	wal, err := NewWAL(dir, true, 1024)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	t.Cleanup(func() {
		if err := wal.Close(); err != nil {
			t.Fatal("expected no error, got", err)
		}
	})

	const workers = 8
	const docsPerWorker = 200
	expectedDocs := workers * docsPerWorker

	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for workerID := 0; workerID < workers; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < docsPerWorker; i++ {
				doc := Document{
					ID:        fmt.Sprintf("%d-%d", workerID, i),
					Embedding: []float32{1, 0, 0},
					Content:   "rotating wal write",
				}

				if err := wal.Append(doc, true, ""); err != nil {
					errCh <- err
					return
				}
			}
		}(workerID)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatal("expected no error, got", err)
	}

	if err := wal.Close(); err != nil {
		t.Fatal("expected no error, got", err)
	}

	walFiles, err := listWALSegments(dir)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if len(walFiles) < 2 {
		t.Fatal("expected rotation with multiple WAL files, got", len(walFiles))
	}

	replayed := &Collection{
		documents: make(map[string]*Document),
	}
	for _, walFile := range walFiles {
		err := wal.replayWAL(walFile, replayed)
		if err != nil {
			t.Fatal("expected no error replaying WAL, got", err)
		}
	}

	if replayed.Count() != expectedDocs {
		t.Fatal("expected", expectedDocs, "replayed documents, got", replayed.Count())
	}
}

func TestDB_WAL_ConfigurableSegmentMaxSize(t *testing.T) {
	path := t.TempDir()

	db, err := NewPersistentDBWithOptions(path, DBConfig{
		Compress:          true,
		Wal:               true,
		WalSegmentMaxSize: 1024,
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	defer db.Close()

	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{1, 0, 0}, nil
	}

	collection, err := db.GetOrCreateCollection("wal-config", nil, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if collection.wal == nil {
		t.Fatal("expected WAL, got nil")
	}
	if collection.wal.maxSegmentSize != 1024 {
		t.Fatal("expected WAL max segment size 1024, got", collection.wal.maxSegmentSize)
	}
}

func TestDB_WAL_Persistence_AcrossRestart(t *testing.T) {
	path := t.TempDir()

	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{1, 0, 0}, nil
	}

	db, err := NewPersistentDBWithOptions(path, DBConfig{
		Compress: true,
		Wal:      true,
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	collection, err := db.GetOrCreateCollection("wal-persist", map[string]string{"k": "v"}, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	const docCount = 300
	docs := make([]Document, 0, docCount)
	for i := 0; i < docCount; i++ {
		docs = append(docs, Document{
			ID:        fmt.Sprintf("doc-%d", i),
			Embedding: []float32{1, 0, 0},
			Content:   fmt.Sprintf("content-%d", i),
		})
	}

	err = collection.AddDocuments(context.Background(), docs, 8)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	if err := db.Close(); err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Re-open and verify documents are recovered from persisted metadata + WAL.
	reopened, err := NewPersistentDBWithOptions(path, DBConfig{
		Compress: true,
		Wal:      true,
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	defer reopened.Close()

	reopenedCollection := reopened.GetCollection("wal-persist", embeddingFunc)
	if reopenedCollection == nil {
		t.Fatal("expected collection, got nil")
	}

	if reopenedCollection.Count() != docCount {
		t.Fatal("expected", docCount, "documents, got", reopenedCollection.Count())
	}

	doc, err := reopenedCollection.GetByID(context.Background(), "doc-42")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if doc.Content != "content-42" {
		t.Fatal("expected content-42, got", doc.Content)
	}
}

func listWALSegments(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	segments := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), walFileExtension) {
			continue
		}
		segments = append(segments, filepath.Join(dir, entry.Name()))
	}

	sort.Strings(segments)
	return segments, nil
}
