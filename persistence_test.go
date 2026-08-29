package chromem

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPersistenceWrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "chromem-go")
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	defer os.RemoveAll(tempDir)

	type s struct {
		Foo string
		Bar []float32
	}
	obj := s{
		Foo: "test",
		Bar: []float32{-0.40824828, 0.40824828, 0.81649655}, // normalized version of `{-0.1, 0.1, 0.2}`
	}

	t.Run("gob", func(t *testing.T) {
		tempFilePath := tempDir + ".gob"
		if err := persistToFile(tempFilePath, obj, false, ""); err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Check if the file exists.
		_, err = os.Stat(tempFilePath)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Read file and decode
		f, err := os.Open(tempFilePath)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}
		defer f.Close()
		d := gob.NewDecoder(f)
		res := s{}
		err = d.Decode(&res)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Compare
		if !reflect.DeepEqual(obj, res) {
			t.Fatalf("expected %+v, got %+v", obj, res)
		}
	})

	t.Run("gob gzipped", func(t *testing.T) {
		tempFilePath := tempDir + ".gob.gz"
		if err := persistToFile(tempFilePath, obj, true, ""); err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Check if the file exists.
		_, err = os.Stat(tempFilePath)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Read file, decompress and decode
		f, err := os.Open(tempFilePath)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}
		defer f.Close()
		gzr, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}
		d := gob.NewDecoder(gzr)
		res := s{}
		err = d.Decode(&res)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Compare
		if !reflect.DeepEqual(obj, res) {
			t.Fatalf("expected %+v, got %+v", obj, res)
		}
	})
}

func TestPersistenceRead(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "chromem-go")
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	defer os.RemoveAll(tempDir)

	type s struct {
		Foo string
		Bar []float32
	}
	obj := s{
		Foo: "test",
		Bar: []float32{-0.40824828, 0.40824828, 0.81649655}, // normalized version of `{-0.1, 0.1, 0.2}`
	}

	t.Run("gob", func(t *testing.T) {
		tempFilePath := tempDir + ".gob"
		f, err := os.Create(tempFilePath)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}
		enc := gob.NewEncoder(f)
		err = enc.Encode(obj)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}
		err = f.Close()
		if err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Read the file.
		var res s
		err = readFromFile(tempFilePath, &res, "")
		if err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Compare
		if !reflect.DeepEqual(obj, res) {
			t.Fatalf("expected %+v, got %+v", obj, res)
		}
	})

	t.Run("gob gzipped", func(t *testing.T) {
		tempFilePath := tempDir + ".gob.gz"
		f, err := os.Create(tempFilePath)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}
		gzw := gzip.NewWriter(f)
		enc := gob.NewEncoder(gzw)
		err = enc.Encode(obj)
		if err != nil {
			t.Fatal("expected nil, got", err)
		}
		err = gzw.Close()
		if err != nil {
			t.Fatal("expected nil, got", err)
		}
		err = f.Close()
		if err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Read the file.
		var res s
		err = readFromFile(tempFilePath, &res, "")
		if err != nil {
			t.Fatal("expected nil, got", err)
		}

		// Compare
		if !reflect.DeepEqual(obj, res) {
			t.Fatalf("expected %+v, got %+v", obj, res)
		}
	})
}

func TestPersistenceEncryption(t *testing.T) {
	// Instead of copy pasting encryption/decryption code, we resort to using both
	// functions under test, instead of one combined with an independent implementation.

	r := rand.New(rand.NewSource(rand.Int63()))
	// randString := randomString(r, 10)
	path := filepath.Join(os.TempDir(), "a", "chromem-go")
	// defer os.RemoveAll(path)

	type s struct {
		Foo string
		Bar []float32
	}
	obj := s{
		Foo: "test",
		Bar: []float32{-0.40824828, 0.40824828, 0.81649655}, // normalized version of `{-0.1, 0.1, 0.2}`
	}
	encryptionKey := randomString(r, 32)

	tt := []struct {
		name     string
		filePath string
		compress bool
	}{
		{
			name:     "compress false",
			filePath: path + ".gob.enc",
			compress: false,
		},
		{
			name:     "compress true",
			filePath: path + ".gob.gz.enc",
			compress: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := persistToFile(tc.filePath, obj, tc.compress, encryptionKey)
			if err != nil {
				t.Fatal("expected nil, got", err)
			}

			// Check if the file exists.
			_, err = os.Stat(tc.filePath)
			if err != nil {
				t.Fatal("expected nil, got", err)
			}

			// Read the file.
			var res s
			err = readFromFile(tc.filePath, &res, encryptionKey)
			if err != nil {
				t.Fatal("expected nil, got", err)
			}

			// Compare
			if !reflect.DeepEqual(obj, res) {
				t.Fatalf("expected %+v, got %+v", obj, res)
			}
		})
	}
}

// findHash2hexCollision deterministically finds two distinct strings whose
// first 4 bytes of sha256 are equal, by hashing candidates "id-0", "id-1",
// ... and looking for the first repeat among the first 4 bytes. With a
// 32-bit truncation this is expected within roughly 2^16 candidates
// (birthday bound).
func findHash2hexCollision(t *testing.T) (string, string) {
	t.Helper()

	seen := make(map[[4]byte]string)
	for i := 0; ; i++ {
		id := fmt.Sprintf("id-%d", i)
		hash := sha256.Sum256([]byte(id))
		var prefix [4]byte
		copy(prefix[:], hash[:4])
		if other, ok := seen[prefix]; ok {
			return other, id
		}
		seen[prefix] = id

		if i > 1<<20 {
			t.Fatal("couldn't find a hash2hex collision within a reasonable number of candidates")
		}
	}
}

// TestPersistentDB_CollidingDocumentFilenamesSurviveReload documents a bug
// where two distinct document IDs whose sha256 hash shares its first 4 bytes
// are persisted to the same on-disk filename, so the second document's file
// silently overwrites the first one's. Both documents must survive a DB
// reload.
func TestPersistentDB_CollidingDocumentFilenamesSurviveReload(t *testing.T) {
	id1, id2 := findHash2hexCollision(t)

	tmpDir, err := os.MkdirTemp(os.TempDir(), "chromem-test-*")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := NewPersistentDB(tmpDir, false)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	name := "test"
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{-0.40824828, 0.40824828, 0.81649655}, nil
	}
	c, err := db.CreateCollection(name, nil, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	err = c.AddDocument(context.Background(), Document{
		ID:      id1,
		Content: "content for " + id1,
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	err = c.AddDocument(context.Background(), Document{
		ID:      id2,
		Content: "content for " + id2,
	})
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Open a fresh DB on the same directory to force a reload from disk.
	reloadedDB, err := NewPersistentDB(tmpDir, false)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	reloadedCollection := reloadedDB.GetCollection(name, embeddingFunc)
	if reloadedCollection == nil {
		t.Fatal("expected collection, got nil")
	}

	if reloadedCollection.Count() != 2 {
		t.Fatalf("expected 2 documents after reload, got %d", reloadedCollection.Count())
	}

	got1, err := reloadedCollection.GetByID(context.Background(), id1)
	if err != nil {
		t.Fatalf("expected document %q to survive reload, got error: %v", id1, err)
	}
	if got1.Content != "content for "+id1 {
		t.Fatalf("expected content %q, got %q", "content for "+id1, got1.Content)
	}

	got2, err := reloadedCollection.GetByID(context.Background(), id2)
	if err != nil {
		t.Fatalf("expected document %q to survive reload, got error: %v", id2, err)
	}
	if got2.Content != "content for "+id2 {
		t.Fatalf("expected content %q, got %q", "content for "+id2, got2.Content)
	}
}
