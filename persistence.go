package chromem

import (
	"bytes"
	"compress/gzip"
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
// as gob, optionally compressed with flate (as gzip) and optionally encrypted with
// AES-GCM. The encryption key must be 32 bytes long. If the file exists, it's
// overwritten, otherwise created.
func persistToFile(filePath string, obj any, compress bool, encryptionKey string) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
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

	return persistToWriter(f, obj, compress, encryptionKey)
}

// persistToWriter persists an object to a writer. The object is serialized
// as gob, optionally compressed with flate (as gzip) and optionally encrypted with
// AES-GCM. The encryption key must be 32 bytes long.
// If the writer has to be closed, it's the caller's responsibility.
func persistToWriter(w io.Writer, obj any, compress bool, encryptionKey string) error {
	// AES 256 requires a 32 byte key
	if encryptionKey != "" {
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// We want to:
	// Encode as gob -> compress with flate -> encrypt with AES-GCM -> write to
	// passed writer.
	// To reduce memory usage we chain the writers instead of buffering, so we start
	// from the end. For AES GCM sealing the stdlib doesn't provide a writer though.

	var chainedWriter io.Writer
	if encryptionKey == "" {
		chainedWriter = w
	} else {
		chainedWriter = &bytes.Buffer{}
	}

	var gzw *gzip.Writer
	var enc *gob.Encoder
	if compress {
		gzw = gzip.NewWriter(chainedWriter)
		enc = gob.NewEncoder(gzw)
	} else {
		enc = gob.NewEncoder(chainedWriter)
	}

	// Start encoding, it will write to the chain of writers.
	if err := enc.Encode(obj); err != nil {
		return fmt.Errorf("couldn't encode or write object: %w", err)
	}

	// If compressing, close the gzip writer. Otherwise, the gzip footer won't be
	// written yet. When using encryption (and chainedWriter is a buffer) then
	// we'll encrypt an incomplete stream. Without encryption when we return here and having
	// a deferred Close(), there might be a silenced error.
	if compress {
		err := gzw.Close()
		if err != nil {
			return fmt.Errorf("couldn't close gzip writer: %w", err)
		}
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

	return readFromReader(r, obj, encryptionKey)
}

// readFromReader reads an object from a Reader. The object is deserialized from gob.
// `obj` must be a pointer to an instantiated object. The stream may optionally
// be compressed as gzip and/or encrypted with AES-GCM. The encryption key must
// be 32 bytes long.
// If the reader has to be closed, it's the caller's responsibility.
func readFromReader(r io.ReadSeeker, obj any, encryptionKey string) error {
	// AES 256 requires a 32 byte key
	if encryptionKey != "" {
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// We want to:
	// Read from reader -> decrypt with AES-GCM -> decompress with flate -> decode
	// as gob.
	// To reduce memory usage we chain the readers instead of buffering, so we start
	// from the end. For the decryption there's no reader though.

	// For the chainedReader we don't declare it as ReadSeeker, so we can reassign
	// the gzip reader to it.
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

	// Determine if the stream is compressed
	magicNumber := make([]byte, 2)
	_, err := chainedReader.Read(magicNumber)
	if err != nil {
		return fmt.Errorf("couldn't read magic number to determine whether the stream is compressed: %w", err)
	}
	var compressed bool
	if magicNumber[0] == 0x1f && magicNumber[1] == 0x8b {
		compressed = true
	}

	// Reset reader. Both the reader from the param and bytes.Reader support seeking.
	if s, ok := chainedReader.(io.Seeker); !ok {
		return fmt.Errorf("reader doesn't support seeking")
	} else {
		_, err := s.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("couldn't reset reader: %w", err)
		}
	}

	if compressed {
		gzr, err := gzip.NewReader(chainedReader)
		if err != nil {
			return fmt.Errorf("couldn't create gzip reader: %w", err)
		}
		defer gzr.Close()
		chainedReader = gzr
	}

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
