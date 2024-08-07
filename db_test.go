package chromem

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestNewPersistentDB(t *testing.T) {
	t.Run("Create directory", func(t *testing.T) {
		r := rand.New(rand.NewSource(rand.Int63()))
		randString := randomString(r, 10)
		path := filepath.Join(os.TempDir(), randString)
		defer os.RemoveAll(path)

		// Path shouldn't exist yet
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("expected path to not exist, got", err)
		}

		db, err := NewPersistentDB(path, false)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if db == nil {
			t.Fatal("expected DB, got nil")
		}

		// Path should exist now
		if _, err := os.Stat(path); err != nil {
			t.Fatal("expected path to exist, got", err)
		}
	})
	t.Run("Existing directory", func(t *testing.T) {
		path, err := os.MkdirTemp(os.TempDir(), "")
		if err != nil {
			t.Fatal("couldn't create temp dir:", err)
		}
		defer os.RemoveAll(path)

		db, err := NewPersistentDB(path, false)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if db == nil {
			t.Fatal("expected DB, got nil")
		}
	})
}

func TestNewPersistentDB_Errors(t *testing.T) {
	t.Run("Path is an existing file", func(t *testing.T) {
		f, err := os.CreateTemp(os.TempDir(), "")
		if err != nil {
			t.Fatal("couldn't create temp file:", err)
		}
		defer os.RemoveAll(f.Name())

		_, err = NewPersistentDB(f.Name(), false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDB_ImportExport(t *testing.T) {
	r := rand.New(rand.NewSource(rand.Int63()))
	randString := randomString(r, 10)
	path := filepath.Join(os.TempDir(), randString)
	defer os.RemoveAll(path)

	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	tt := []struct {
		name          string
		filePath      string
		compress      bool
		encryptionKey string
	}{
		{
			name:          "gob",
			filePath:      path + ".gob",
			compress:      false,
			encryptionKey: "",
		},
		{
			name:          "gob compressed",
			filePath:      path + ".gob.gz",
			compress:      true,
			encryptionKey: "",
		},
		{
			name:          "gob compressed encrypted",
			filePath:      path + ".gob.gz.enc",
			compress:      true,
			encryptionKey: randomString(r, 32),
		},
		{
			name:          "gob encrypted",
			filePath:      path + ".gob.enc",
			compress:      false,
			encryptionKey: randomString(r, 32),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Create DB, can just be in-memory
			origDB := NewDB()

			// Create collection
			c, err := origDB.CreateCollection(name, metadata, embeddingFunc)
			if err != nil {
				t.Fatal("expected no error, got", err)
			}
			if c == nil {
				t.Fatal("expected collection, got nil")
			}
			// Add document
			doc := Document{
				ID:        name,
				Metadata:  metadata,
				Embedding: vectors,
				Content:   "test",
			}
			err = c.AddDocument(context.Background(), doc)
			if err != nil {
				t.Fatal("expected no error, got", err)
			}

			// Export
			err = origDB.ExportToFile(tc.filePath, tc.compress, tc.encryptionKey)
			if err != nil {
				t.Fatal("expected no error, got", err)
			}

			newDB := NewDB()

			// Import
			err = newDB.ImportFromFile(tc.filePath, tc.encryptionKey)
			if err != nil {
				t.Fatal("expected no error, got", err)
			}

			// Check expectations
			// We have to reset the embed function, but otherwise the DB objects
			// should be deep equal.
			c.embed = nil
			if !reflect.DeepEqual(origDB, newDB) {
				t.Fatalf("expected DB %+v, got %+v", origDB, newDB)
			}
		})
	}
}

func TestDB_ImportExportSpecificCollections(t *testing.T) {
	r := rand.New(rand.NewSource(rand.Int63()))
	randString := randomString(r, 10)
	path := filepath.Join(os.TempDir(), randString)
	filePath := path + ".gob"
	defer os.RemoveAll(path)

	// Values in the collection
	name := "test"
	name2 := "test2"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create DB, can just be in-memory
	origDB := NewDB()

	// Create collections
	c, err := origDB.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	c2, err := origDB.CreateCollection(name2, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Add documents
	doc := Document{
		ID:        name,
		Metadata:  metadata,
		Embedding: vectors,
		Content:   "test",
	}

	doc2 := Document{
		ID:        name2,
		Metadata:  metadata,
		Embedding: vectors,
		Content:   "test2",
	}

	err = c.AddDocument(context.Background(), doc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	err = c2.AddDocument(context.Background(), doc2)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Export only one of the two collections
	err = origDB.ExportToFile(filePath, false, "", name2)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	dir := filepath.Join(path, randomString(r, 10))
	defer os.RemoveAll(dir)

	// Instead of importing to an in-memory DB we use a persistent one to cover the behavior of immediate persistent files being created for the imported data
	newPDB, err := NewPersistentDB(dir, false)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	err = newPDB.ImportFromFile(filePath, "")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	if len(newPDB.collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(newPDB.collections))
	}

	// Make sure that the imported documents are actually persisted on disk
	for _, col := range newPDB.collections {
		for _, d := range col.documents {
			_, err = os.Stat(col.getDocPath(d.ID))
			if err != nil {
				t.Fatalf("expected no error when looking up persistent file for doc %q, got %v", d.ID, err)
			}
		}
	}

	// Now export both collections and import them into the same persistent DB (overwriting the one we just imported)
	filePath2 := filepath.Join(path, "2.gob")
	err = origDB.ExportToFile(filePath2, false, "")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	err = newPDB.ImportFromFile(filePath2, "")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	if len(newPDB.collections) != 2 {
		t.Fatalf("expected 2 collections, got %d", len(newPDB.collections))
	}

	// Make sure that the imported documents are actually persisted on disk
	for _, col := range newPDB.collections {
		for _, d := range col.documents {
			_, err = os.Stat(col.getDocPath(d.ID))
			if err != nil {
				t.Fatalf("expected no error when looking up persistent file for doc %q, got %v", d.ID, err)
			}
		}
	}
}

func TestDB_CreateCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	db := NewDB()

	t.Run("OK", func(t *testing.T) {
		c, err := db.CreateCollection(name, metadata, embeddingFunc)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if c == nil {
			t.Fatal("expected collection, got nil")
		}

		// Check expectations

		// DB should have one collection now
		if len(db.collections) != 1 {
			t.Fatal("expected 1 collection, got", len(db.collections))
		}
		// The collection should be the one we just created
		c2, ok := db.collections[name]
		if !ok {
			t.Fatal("expected collection", name, "not found")
		}
		// Check the embedding function first, then the rest with DeepEqual
		gotVectors, err := c.embed(context.Background(), "test")
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if !slices.Equal(gotVectors, vectors) {
			t.Fatal("expected vectors", vectors, "got", gotVectors)
		}
		c.embed, c2.embed = nil, nil
		if !reflect.DeepEqual(c, c2) {
			t.Fatalf("expected collection %+v, got %+v", c, c2)
		}
	})

	t.Run("NOK - Empty name", func(t *testing.T) {
		_, err := db.CreateCollection("", metadata, embeddingFunc)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDB_ListCollections(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	orig, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// List collections
	res := db.ListCollections()

	// Check expectations

	// Should've returned a map with one collection
	if len(res) != 1 {
		t.Fatal("expected 1 collection, got", len(res))
	}
	// The collection should be the one we just created
	c, ok := res[name]
	if !ok {
		t.Fatal("expected collection", name, "not found")
	}
	// Check the embedding function first, then the rest with DeepEqual
	gotVectors, err := c.embed(context.Background(), "test")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if !slices.Equal(gotVectors, vectors) {
		t.Fatal("expected vectors", vectors, "got", gotVectors)
	}
	orig.embed, c.embed = nil, nil
	if !reflect.DeepEqual(orig, c) {
		t.Fatalf("expected collection %+v, got %+v", orig, c)
	}

	// And it should be a copy. Adding a value here should not reflect on the DB's
	// collection.
	res["foo"] = &Collection{}
	if len(db.ListCollections()) != 1 {
		t.Fatal("expected 1 collection, got", len(db.ListCollections()))
	}
}

func TestDB_GetCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	orig, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Get collection
	c := db.GetCollection(name, nil)

	// Check the embedding function first, then the rest with DeepEqual
	gotVectors, err := c.embed(context.Background(), "test")
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if !slices.Equal(gotVectors, vectors) {
		t.Fatal("expected vectors", vectors, "got", gotVectors)
	}
	orig.embed, c.embed = nil, nil
	if !reflect.DeepEqual(orig, c) {
		t.Fatalf("expected collection %+v, got %+v", orig, c)
	}
}

func TestDB_GetOrCreateCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	t.Run("Get", func(t *testing.T) {
		// Create initial collection
		db := NewDB()
		// Create collection so that the GetOrCreateCollection() call below only
		// gets it.
		orig, err := db.CreateCollection(name, metadata, embeddingFunc)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}

		// Call GetOrCreateCollection() with the same name to only get it. We pass
		// nil for the metadata and embeddingFunc, so we can check that the returned
		// collection is the original one, and not a new one.
		c, err := db.GetOrCreateCollection(name, nil, nil)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if c == nil {
			t.Fatal("expected collection, got nil")
		}

		// Check the embedding function first, then the rest with DeepEqual
		gotVectors, err := c.embed(context.Background(), "test")
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if !slices.Equal(gotVectors, vectors) {
			t.Fatal("expected vectors", vectors, "got", gotVectors)
		}
		orig.embed, c.embed = nil, nil
		if !reflect.DeepEqual(orig, c) {
			t.Fatalf("expected collection %+v, got %+v", orig, c)
		}
	})

	t.Run("Create", func(t *testing.T) {
		// Create initial collection
		db := NewDB()

		// Call GetOrCreateCollection()
		c, err := db.GetOrCreateCollection(name, metadata, embeddingFunc)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if c == nil {
			t.Fatal("expected collection, got nil")
		}

		// Check like we check CreateCollection()
		c2, ok := db.collections[name]
		if !ok {
			t.Fatal("expected collection", name, "not found")
		}
		gotVectors, err := c.embed(context.Background(), "test")
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if !slices.Equal(gotVectors, vectors) {
			t.Fatal("expected vectors", vectors, "got", gotVectors)
		}
		c.embed, c2.embed = nil, nil
		if !reflect.DeepEqual(c, c2) {
			t.Fatalf("expected collection %+v, got %+v", c, c2)
		}
	})
}

func TestDB_DeleteCollection(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Delete collection
	if err := db.DeleteCollection(name); err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Check expectations
	// We don't have access to the documents field, but we can rely on DB.ListCollections()
	// because it's tested elsewhere.
	if len(db.ListCollections()) != 0 {
		t.Fatal("expected 0 collections, got", len(db.ListCollections()))
	}
	// Also check internally
	if len(db.collections) != 0 {
		t.Fatal("expected 0 collections, got", len(db.collections))
	}
}

func TestDB_Reset(t *testing.T) {
	// Values in the collection
	name := "test"
	metadata := map[string]string{"foo": "bar"}
	vectors := []float32{-0.40824828, 0.40824828, 0.81649655} // normalized version of `{-0.1, 0.1, 0.2}`
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	// Create initial collection
	db := NewDB()
	// We ignore the return value. CreateCollection is tested elsewhere.
	_, err := db.CreateCollection(name, metadata, embeddingFunc)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Reset DB
	if err := db.Reset(); err != nil {
		t.Fatal("expected no error, got", err)
	}

	// Check expectations
	// We don't have access to the documents field, but we can rely on DB.ListCollections()
	// because it's tested elsewhere.
	if len(db.ListCollections()) != 0 {
		t.Fatal("expected 0 collections, got", len(db.ListCollections()))
	}
	// Also check internally
	if len(db.collections) != 0 {
		t.Fatal("expected 0 collections, got", len(db.collections))
	}
}
