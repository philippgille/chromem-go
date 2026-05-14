package chromem

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRegisterCompression(t *testing.T) {
	codec := testCompressionCodec(".tc1", []byte("tc1"))
	if err := RegisterCompression("test-register", codec); err != nil {
		t.Fatal("expected nil, got", err)
	}
	if err := RegisterCompression("test-register", testCompressionCodec(".tc2", []byte("tc2"))); err == nil {
		t.Fatal("expected duplicate name error, got nil")
	}
	if err := RegisterCompression(CompressionGzip, testCompressionCodec(".tc3", []byte("tc3"))); err == nil {
		t.Fatal("expected reserved name error, got nil")
	}
	if err := RegisterCompression("test-bad-extension", CompressionCodec{
		Extension:   "bad",
		MagicNumber: []byte("tc4"),
		NewWriter:   codec.NewWriter,
		NewReader:   codec.NewReader,
	}); err == nil {
		t.Fatal("expected invalid extension error, got nil")
	}
	if err := RegisterCompression("test-duplicate-extension", CompressionCodec{
		Extension:   ".tc1",
		MagicNumber: []byte("tc5"),
		NewWriter:   codec.NewWriter,
		NewReader:   codec.NewReader,
	}); err == nil {
		t.Fatal("expected duplicate extension error, got nil")
	}
	if err := RegisterCompression("test-overlapping-magic", CompressionCodec{
		Extension:   ".test-overlapping-magic",
		MagicNumber: []byte("tc1-overlap"),
		NewWriter:   codec.NewWriter,
		NewReader:   codec.NewReader,
	}); err == nil {
		t.Fatal("expected overlapping magic error, got nil")
	}
}

func TestCompressionCodecRoundTrip(t *testing.T) {
	compression := Compression("test-roundtrip")
	if err := RegisterCompression(compression, testCompressionCodec(".roundtrip", []byte("tr1"))); err != nil {
		t.Fatal("expected nil, got", err)
	}

	type s struct {
		Foo string
		Bar []float32
	}
	obj := s{
		Foo: "test",
		Bar: []float32{-0.40824828, 0.40824828, 0.81649655},
	}

	filePath := filepath.Join(t.TempDir(), "test.gob.roundtrip")
	if err := persistToFileWithCompression(filePath, obj, compression, ""); err != nil {
		t.Fatal("expected nil, got", err)
	}
	var got s
	if err := readFromFile(filePath, &got, ""); err != nil {
		t.Fatal("expected nil, got", err)
	}
	if !reflect.DeepEqual(obj, got) {
		t.Fatalf("expected %+v, got %+v", obj, got)
	}
}

func TestDBExportWithRegisteredCompression(t *testing.T) {
	compression := Compression("test-db-export")
	if err := RegisterCompression(compression, testCompressionCodec(".db-export", []byte("tdb"))); err != nil {
		t.Fatal("expected nil, got", err)
	}

	vectors := []float32{-0.40824828, 0.40824828, 0.81649655}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	origDB := NewDB()
	c, err := origDB.CreateCollection("test", map[string]string{"foo": "bar"}, embeddingFunc)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	if err := c.AddDocument(context.Background(), Document{
		ID:        "doc1",
		Metadata:  map[string]string{"foo": "bar"},
		Embedding: vectors,
		Content:   "test",
	}); err != nil {
		t.Fatal("expected nil, got", err)
	}

	var buf bytes.Buffer
	if err := origDB.ExportToWriterWithCompression(&buf, compression, ""); err != nil {
		t.Fatal("expected nil, got", err)
	}

	newDB := NewDB()
	if err := newDB.ImportFromReader(bytes.NewReader(buf.Bytes()), ""); err != nil {
		t.Fatal("expected nil, got", err)
	}

	c.embed = nil
	if !reflect.DeepEqual(origDB, newDB) {
		t.Fatalf("expected DB %+v, got %+v", origDB, newDB)
	}
}

func TestDBImportFromReaderWithCompression(t *testing.T) {
	compression := Compression("test-import-explicit")
	if err := RegisterCompression(compression, testCompressionCodec(".import-explicit", []byte("imp"))); err != nil {
		t.Fatal("expected nil, got", err)
	}

	vectors := []float32{-0.40824828, 0.40824828, 0.81649655}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	origDB := NewDB()
	c, err := origDB.CreateCollection("test", map[string]string{"foo": "bar"}, embeddingFunc)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	if err := c.AddDocument(context.Background(), Document{
		ID:        "doc1",
		Metadata:  map[string]string{"foo": "bar"},
		Embedding: vectors,
		Content:   "test",
	}); err != nil {
		t.Fatal("expected nil, got", err)
	}

	var buf bytes.Buffer
	if err := origDB.ExportToWriterWithCompression(&buf, compression, ""); err != nil {
		t.Fatal("expected nil, got", err)
	}

	newDB := NewDB()
	if err := newDB.ImportFromReaderWithCompression(bytes.NewReader(buf.Bytes()), compression, ""); err != nil {
		t.Fatal("expected nil, got", err)
	}
	c.embed = nil
	if !reflect.DeepEqual(origDB, newDB) {
		t.Fatalf("expected DB %+v, got %+v", origDB, newDB)
	}

	// Unknown compression name should error.
	if err := NewDB().ImportFromReaderWithCompression(bytes.NewReader(buf.Bytes()), "no-such-codec", ""); err == nil {
		t.Fatal("expected error for unknown compression, got nil")
	}
}

func TestPersistentDBWithRegisteredCompression(t *testing.T) {
	compression := Compression("test-persistent")
	if err := RegisterCompression(compression, testCompressionCodec(".test-persistent", nil)); err != nil {
		t.Fatal("expected nil, got", err)
	}

	vectors := []float32{-0.40824828, 0.40824828, 0.81649655}
	embeddingFunc := func(_ context.Context, _ string) ([]float32, error) {
		return vectors, nil
	}

	dir := t.TempDir()
	db, err := NewPersistentDBWithCompression(dir, compression)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	c, err := db.CreateCollection("test", map[string]string{"foo": "bar"}, embeddingFunc)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	doc := Document{
		ID:        "doc1",
		Metadata:  map[string]string{"foo": "bar"},
		Embedding: vectors,
		Content:   "test",
	}
	if err := c.AddDocument(context.Background(), doc); err != nil {
		t.Fatal("expected nil, got", err)
	}
	if _, err := os.Stat(filepath.Join(c.persistDirectory, metadataFileName+".gob.test-persistent")); err != nil {
		t.Fatal("expected metadata file to exist, got", err)
	}

	loadedDB, err := NewPersistentDBWithCompression(dir, compression)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	loadedCollection := loadedDB.GetCollection("test", embeddingFunc)
	if loadedCollection == nil {
		t.Fatal("expected collection, got nil")
	}
	got, err := loadedCollection.GetByID(context.Background(), doc.ID)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	if !reflect.DeepEqual(doc, got) {
		t.Fatalf("expected %+v, got %+v", doc, got)
	}
}

func testCompressionCodec(extension string, magic []byte) CompressionCodec {
	return CompressionCodec{
		Extension:   extension,
		MagicNumber: magic,
		NewWriter: func(w io.Writer) (io.WriteCloser, error) {
			return &testCompressionWriter{w: w, magic: magic}, nil
		},
		NewReader: func(r io.Reader) (io.ReadCloser, error) {
			if len(magic) > 0 {
				got := make([]byte, len(magic))
				if _, err := io.ReadFull(r, got); err != nil {
					return nil, err
				}
				if !bytes.Equal(got, magic) {
					return nil, fmt.Errorf("unexpected magic number %q", got)
				}
			}
			return io.NopCloser(r), nil
		},
	}
}

type testCompressionWriter struct {
	w     io.Writer
	magic []byte
	wrote bool
}

func (w *testCompressionWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.wrote = true
		if _, err := w.w.Write(w.magic); err != nil {
			return 0, err
		}
	}
	return w.w.Write(p)
}

func (w *testCompressionWriter) Close() error {
	return nil
}
