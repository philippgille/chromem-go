package main

import (
	"context"
	"runtime"
	"strconv"
	"testing"

	chromem "github.com/philippgille/chromem-go"
)

const recordCount = 1000

func fakeEmbeddingFunc(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, 384)
	for i := range vec {
		vec[i] = float32(i) * 0.001
	}
	return vec, nil
}

func generateDocs(n int) []chromem.Document {
	docs := make([]chromem.Document, n)
	for i := 0; i < n; i++ {
		docs[i] = chromem.Document{
			ID:      strconv.Itoa(i),
			Content: "The sky is blue because of Rayleigh scattering.",
		}
	}
	return docs
}

func benchmarkAddDocuments(b *testing.B, useWAL bool) {

	ctx := context.Background()
	dbPath := b.TempDir()

	var db *chromem.DB
	var err error

	if useWAL {
		db, err = chromem.NewPersistentDBWithOptions(dbPath, chromem.DBConfig{
			Compress: true,
			Wal:      true,
		})
	} else {
		db, err = chromem.NewPersistentDB(dbPath, true)
	}

	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := db.Close(); err != nil {
			b.Fatalf("couldn't close DB: %v", err)
		}
	})

	collection, err := db.GetOrCreateCollection("bench", nil, fakeEmbeddingFunc)
	if err != nil {
		b.Fatal(err)
	}

	docs := generateDocs(recordCount)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		// Make IDs unique for every iteration
		for j := range docs {
			docs[j].ID = strconv.Itoa(i*recordCount + j)
		}

		err := collection.AddDocuments(ctx, docs, runtime.NumCPU())
		if err != nil {
			b.Fatal(err)
		}

		// time.Sleep(2 * time.Second)
	}
}

func BenchmarkAddDocuments_10k(b *testing.B) {
	benchmarkAddDocuments(b, false)
}

func BenchmarkAddDocumentsUsingWAL_10k(b *testing.B) {
	benchmarkAddDocuments(b, true)
}
