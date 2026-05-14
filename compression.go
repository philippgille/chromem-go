package chromem

import (
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Compression identifies the compression algorithm used for persistence and
// export files.
//
// The zero value (empty string) has context-dependent meaning:
//   - Write paths (Export, NewPersistentDB) treat it as [CompressionNone].
//   - Read paths that accept an optional compression (e.g.
//     [DB.ImportFromFileWithCompression]) treat it as "auto-detect from magic
//     bytes". Auto-detect only works for codecs that declare a non-empty
//     [CompressionCodec.MagicNumber].
type Compression string

const (
	// CompressionNone disables compression.
	CompressionNone Compression = "none"

	// CompressionGzip compresses files with gzip.
	CompressionGzip Compression = "gzip"
)

// CompressionCodec describes how to compress and decompress a stream for a
// given Compression name. External codecs register themselves from client code.
type CompressionCodec struct {
	// Extension is appended to ".gob" for on-disk files. It must be empty for
	// no compression or start with a dot, for example ".zst".
	Extension string

	// MagicNumber is used to auto-detect the codec on the read path. Codecs with
	// an empty magic number are not eligible for auto-detection.
	MagicNumber []byte

	// NewWriter wraps w. The returned WriteCloser must flush all bytes on Close.
	NewWriter func(w io.Writer) (io.WriteCloser, error)

	// NewReader wraps r. The returned ReadCloser must be safe to Close once.
	NewReader func(r io.Reader) (io.ReadCloser, error)
}

var compressionRegistry = struct {
	sync.RWMutex
	codecs      map[Compression]CompressionCodec
	maxMagicLen int
}{
	codecs: make(map[Compression]CompressionCodec),
}

func init() {
	mustRegisterCompression(CompressionNone, CompressionCodec{
		NewWriter: func(w io.Writer) (io.WriteCloser, error) {
			return nopWriteCloser{Writer: w}, nil
		},
		NewReader: func(r io.Reader) (io.ReadCloser, error) {
			return io.NopCloser(r), nil
		},
	})
	mustRegisterCompression(CompressionGzip, CompressionCodec{
		Extension:   ".gz",
		MagicNumber: []byte{0x1f, 0x8b},
		NewWriter: func(w io.Writer) (io.WriteCloser, error) {
			return gzip.NewWriter(w), nil
		},
		NewReader: func(r io.Reader) (io.ReadCloser, error) {
			return gzip.NewReader(r)
		},
	})
}

// RegisterCompression registers a codec under the given name. The names "none"
// and "gzip" are reserved for built-in codecs.
func RegisterCompression(name Compression, codec CompressionCodec) error {
	if name == "" {
		return fmt.Errorf("compression name is empty")
	}
	if name == CompressionNone || name == CompressionGzip {
		return fmt.Errorf("compression name %q is reserved", name)
	}
	return registerCompression(name, codec)
}

func mustRegisterCompression(name Compression, codec CompressionCodec) {
	if err := registerCompression(name, codec); err != nil {
		panic(err)
	}
}

func registerCompression(name Compression, codec CompressionCodec) error {
	if err := validateCompressionCodec(codec); err != nil {
		return err
	}

	codec.MagicNumber = append([]byte(nil), codec.MagicNumber...)

	compressionRegistry.Lock()
	defer compressionRegistry.Unlock()

	if _, ok := compressionRegistry.codecs[name]; ok {
		return fmt.Errorf("compression %q is already registered", name)
	}
	for registeredName, registeredCodec := range compressionRegistry.codecs {
		if codec.Extension == registeredCodec.Extension {
			return fmt.Errorf("compression extension %q is already registered for %q", codec.Extension, registeredName)
		}
		if magicNumbersOverlap(codec.MagicNumber, registeredCodec.MagicNumber) {
			return fmt.Errorf("compression magic number for %q overlaps with %q", name, registeredName)
		}
	}

	compressionRegistry.codecs[name] = codec
	if len(codec.MagicNumber) > compressionRegistry.maxMagicLen {
		compressionRegistry.maxMagicLen = len(codec.MagicNumber)
	}
	return nil
}

func validateCompressionCodec(codec CompressionCodec) error {
	if codec.Extension != "" && !strings.HasPrefix(codec.Extension, ".") {
		return fmt.Errorf("compression extension %q must start with a dot", codec.Extension)
	}
	if codec.NewWriter == nil {
		return fmt.Errorf("compression writer constructor is nil")
	}
	if codec.NewReader == nil {
		return fmt.Errorf("compression reader constructor is nil")
	}
	return nil
}

func magicNumbersOverlap(a, b []byte) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	if len(a) <= len(b) {
		return string(a) == string(b[:len(a)])
	}
	return string(b) == string(a[:len(b)])
}

func lookupCompressionCodec(compression Compression) (CompressionCodec, bool) {
	if compression == "" {
		compression = CompressionNone
	}
	compressionRegistry.RLock()
	defer compressionRegistry.RUnlock()
	codec, ok := compressionRegistry.codecs[compression]
	return codec, ok
}

func detectCompressionCodec(prefix []byte) (CompressionCodec, Compression, bool) {
	compressionRegistry.RLock()
	defer compressionRegistry.RUnlock()
	for name, codec := range compressionRegistry.codecs {
		if len(codec.MagicNumber) == 0 || len(prefix) < len(codec.MagicNumber) {
			continue
		}
		if string(prefix[:len(codec.MagicNumber)]) == string(codec.MagicNumber) {
			return codec, name, true
		}
	}
	codec, ok := compressionRegistry.codecs[CompressionNone]
	return codec, CompressionNone, ok
}

func maxCompressionMagicLen() int {
	compressionRegistry.RLock()
	defer compressionRegistry.RUnlock()
	return compressionRegistry.maxMagicLen
}

func compressionFromBool(compress bool) Compression {
	if compress {
		return CompressionGzip
	}
	return CompressionNone
}

func (c Compression) validate() error {
	if _, ok := lookupCompressionCodec(c); !ok {
		return fmt.Errorf("unsupported compression: %q", c)
	}
	return nil
}

func (c Compression) fileExtension() (string, error) {
	codec, ok := lookupCompressionCodec(c)
	if !ok {
		return "", fmt.Errorf("unsupported compression: %q", c)
	}
	return ".gob" + codec.Extension, nil
}

type nopWriteCloser struct {
	io.Writer
}

func (n nopWriteCloser) Close() error {
	return nil
}
