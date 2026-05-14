package chromem

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const metadataFileName = "00000000"

func hash2hex(name string) string {
	hash := sha256.Sum256([]byte(name))
	// We encode 4 of the 32 bytes (32 out of 256 bits), so 8 hex characters.
	// It's enough to avoid collisions in reasonable amounts of documents per collection
	// and being shorter is better for file paths.
	return hex.EncodeToString(hash[:4])
}

// persistToFile persists an object to a file at the given path. The object is serialized
// as gob, optionally compressed with gzip and optionally encrypted with
// AES-GCM. The encryption key must be 32 bytes long. If the file exists, it's
// overwritten, otherwise created.
func persistToFile(filePath string, obj any, compress bool, encryptionKey string) error {
	return persistToFileWithCompression(filePath, obj, compressionFromBool(compress), encryptionKey)
}

// persistToFileWithCompression persists an object with a registered codec.
func persistToFileWithCompression(filePath string, obj any, compression Compression, encryptionKey string) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}
	if err := compression.validate(); err != nil {
		return err
	}
	// AES 256 requires a 32 byte key
	if encryptionKey != "" {
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// If path doesn't exist, create the parent path.
	// If path exists, and it's a directory, return an error.
	fi, err := os.Stat(filePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("couldn't get info about the path: %w", err)
		} else {
			// If the file doesn't exist, create the parent path
			err := os.MkdirAll(filepath.Dir(filePath), 0o700)
			if err != nil {
				return fmt.Errorf("couldn't create parent directories to path: %w", err)
			}
		}
	} else if fi.IsDir() {
		return fmt.Errorf("path is a directory: %s", filePath)
	}

	// Open file for writing
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("couldn't create file: %w", err)
	}
	defer f.Close()

	return persistToWriterWithCompression(f, obj, compression, encryptionKey)
}

// persistToWriter persists an object to a writer. The object is serialized
// as gob, optionally compressed with gzip and optionally encrypted with
// AES-GCM. The encryption key must be 32 bytes long.
// If the writer has to be closed, it's the caller's responsibility.
func persistToWriter(w io.Writer, obj any, compress bool, encryptionKey string) error {
	return persistToWriterWithCompression(w, obj, compressionFromBool(compress), encryptionKey)
}

// persistToWriterWithCompression persists an object with a registered codec.
func persistToWriterWithCompression(w io.Writer, obj any, compression Compression, encryptionKey string) error {
	codec, ok := lookupCompressionCodec(compression)
	if !ok {
		return fmt.Errorf("unsupported compression: %q", compression)
	}
	// AES 256 requires a 32 byte key
	if encryptionKey != "" {
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// We want to:
	// Encode as gob -> compress -> encrypt with AES-GCM -> write to
	// passed writer.
	// To reduce memory usage we chain the writers instead of buffering, so we start
	// from the end. For AES GCM sealing the stdlib doesn't provide a writer though.

	var chainedWriter io.Writer
	if encryptionKey == "" {
		chainedWriter = w
	} else {
		chainedWriter = &bytes.Buffer{}
	}

	compressor, err := codec.NewWriter(chainedWriter)
	if err != nil {
		return fmt.Errorf("couldn't create compression writer: %w", err)
	}
	enc := gob.NewEncoder(compressor)

	// Start encoding, it will write to the chain of writers.
	if err := enc.Encode(obj); err != nil {
		_ = compressor.Close()
		return fmt.Errorf("couldn't encode or write object: %w", err)
	}

	// Close before encryption so compression footers are included.
	if err := compressor.Close(); err != nil {
		return fmt.Errorf("couldn't close compression writer: %w", err)
	}

	// Without encyrption, the chain is done and the writing is finished.
	if encryptionKey == "" {
		return nil
	}

	// Otherwise, encrypt and then write to the unchained target writer.
	block, err := aes.NewCipher([]byte(encryptionKey))
	if err != nil {
		return fmt.Errorf("couldn't create new AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("couldn't create GCM wrapper: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("couldn't read random bytes for nonce: %w", err)
	}
	// chainedWriter is a *bytes.Buffer
	buf := chainedWriter.(*bytes.Buffer)
	encrypted := gcm.Seal(nonce, nonce, buf.Bytes(), nil)
	_, err = w.Write(encrypted)
	if err != nil {
		return fmt.Errorf("couldn't write encrypted data: %w", err)
	}

	return nil
}

// readFromFile reads an object from a file at the given path. The object is deserialized
// from gob. `obj` must be a pointer to an instantiated object. The file may
// optionally be compressed as gzip and/or encrypted with AES-GCM. The encryption
// key must be 32 bytes long.
func readFromFile(filePath string, obj any, encryptionKey string) error {
	return readFromFileWithCompression(filePath, obj, encryptionKey, "")
}

func readFromFileWithCompression(filePath string, obj any, encryptionKey string, compression Compression) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}
	// AES 256 requires a 32 byte key
	if encryptionKey != "" {
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	r, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("couldn't open file: %w", err)
	}
	defer r.Close()

	return readFromReaderWithCompression(r, obj, encryptionKey, compression)
}

// readFromReader reads an object from a Reader. The object is deserialized from gob.
// `obj` must be a pointer to an instantiated object. The stream may optionally
// be compressed as gzip and/or encrypted with AES-GCM. The encryption key must
// be 32 bytes long.
// If the reader has to be closed, it's the caller's responsibility.
func readFromReader(r io.ReadSeeker, obj any, encryptionKey string) error {
	return readFromReaderWithCompression(r, obj, encryptionKey, "")
}

func readFromReaderWithCompression(r io.ReadSeeker, obj any, encryptionKey string, compression Compression) error {
	// AES 256 requires a 32 byte key
	if encryptionKey != "" {
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// We want to:
	// Read from reader -> decrypt with AES-GCM -> decompress -> decode
	// as gob.
	// To reduce memory usage we chain the readers instead of buffering, so we start
	// from the end. For the decryption there's no reader though.

	// For the chainedReader we don't declare it as ReadSeeker, so we can reassign
	// compression readers to it.
	var chainedReader io.Reader

	// Decrypt if an encryption key is provided
	if encryptionKey != "" {
		encrypted, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("couldn't read from reader: %w", err)
		}
		block, err := aes.NewCipher([]byte(encryptionKey))
		if err != nil {
			return fmt.Errorf("couldn't create AES cipher: %w", err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return fmt.Errorf("couldn't create GCM wrapper: %w", err)
		}
		nonceSize := gcm.NonceSize()
		if len(encrypted) < nonceSize {
			return fmt.Errorf("encrypted data too short")
		}
		nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
		data, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return fmt.Errorf("couldn't decrypt data: %w", err)
		}

		chainedReader = bytes.NewReader(data)
	} else {
		chainedReader = r
	}

	var codec CompressionCodec
	var ok bool
	if compression == "" {
		magicNumber := make([]byte, maxCompressionMagicLen())
		n, err := io.ReadFull(chainedReader, magicNumber)
		if err != nil && !(errors.Is(err, io.ErrUnexpectedEOF) && n > 0) {
			return fmt.Errorf("couldn't read magic number to determine whether the stream is compressed: %w", err)
		}
		magicNumber = magicNumber[:n]
		chainedReader = io.MultiReader(bytes.NewReader(magicNumber), chainedReader)

		codec, compression, ok = detectCompressionCodec(magicNumber)
		if !ok {
			return fmt.Errorf("unsupported compression: %q", compression)
		}
	} else {
		codec, ok = lookupCompressionCodec(compression)
		if !ok {
			return fmt.Errorf("unsupported compression: %q", compression)
		}
	}

	compressionReader, err := codec.NewReader(chainedReader)
	if err != nil {
		return fmt.Errorf("couldn't create compression reader: %w", err)
	}
	defer compressionReader.Close()
	chainedReader = compressionReader

	dec := gob.NewDecoder(chainedReader)
	err = dec.Decode(obj)
	if err != nil {
		return fmt.Errorf("couldn't decode object: %w", err)
	}

	return nil
}

// removeFile removes a file at the given path. If the file doesn't exist, it's a no-op.
func removeFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}

	err := os.Remove(filePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("couldn't remove file %q: %w", filePath, err)
		}
	}

	return nil
}
