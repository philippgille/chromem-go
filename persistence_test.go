package chromem

import (
	"bytes"
	"encoding/gob"
	"os"
	"reflect"
	"testing"
)

func TestPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "chromem-go")
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	type s struct {
		Foo string
		Bar []float32
	}
	obj := s{
		Foo: "test",
		Bar: []float32{-0.40824828, 0.40824828, 0.81649655}, // normalized version of `{-0.1, 0.1, 0.2}`
	}

	persist(tempDir, obj)

	// Check if the file exists.
	_, err = os.Stat(tempDir + ".gob")
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	// Check if the file contains the expected data.
	b, err := os.ReadFile(tempDir + ".gob")
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	d := gob.NewDecoder(bytes.NewReader(b))
	res := s{}
	err = d.Decode(&res)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	if !reflect.DeepEqual(obj, res) {
		t.Fatalf("expected %+v, got %+v", obj, res)
	}
}
