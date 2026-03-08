package chromem

import (
	"context"
	"runtime"
	"strconv"
	"testing"
)

const walBenchmarkRecordCount = 1000

func walBenchmarkEmbeddingFunc(_ context.Context, _ string) ([]float32, error) {
	vec := make([]float32, 384)
	for i := range vec {
		vec[i] = float32(i) * 0.001
	}
	return vec, nil
}

func walBenchmarkGenerateDocs(n int) []Document {
	docs := make([]Document, n)
	for i := 0; i < n; i++ {
		docs[i] = Document{
			ID:      strconv.Itoa(i),
			Content: "The sky is blue because of Rayleigh scattering.",
		}
	}
	return docs
}

func benchmarkCollectionAddDocuments(b *testing.B, useWAL bool) {
	ctx := context.Background()
	dbPath := b.TempDir()

	var (
		db  *DB
		err error
	)

	if useWAL {
		db, err = NewPersistentDBWithOptions(dbPath, DBConfig{
			Compress: true,
			Wal:      true,
		})
	} else {
		db, err = NewPersistentDB(dbPath, true)
	}
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := db.Close(); err != nil {
			b.Fatalf("couldn't close DB: %v", err)
		}
	})

	collection, err := db.GetOrCreateCollection("bench", nil, walBenchmarkEmbeddingFunc)
	if err != nil {
		b.Fatal(err)
	}

	docs := walBenchmarkGenerateDocs(walBenchmarkRecordCount)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Make IDs unique for every benchmark iteration.
		for j := range docs {
			docs[j].ID = strconv.Itoa(i*walBenchmarkRecordCount + j)
		}

		if err := collection.AddDocuments(ctx, docs, runtime.NumCPU()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCollection_AddDocuments_1k(b *testing.B) {
	benchmarkCollectionAddDocuments(b, false)
}

func BenchmarkCollection_AddDocuments_WAL_1k(b *testing.B) {
	benchmarkCollectionAddDocuments(b, true)
}
