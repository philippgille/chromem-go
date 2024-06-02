package chromem

import (
	"compress/gzip"
	"encoding/gob"
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
