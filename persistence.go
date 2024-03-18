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

// persist persists an object to a file at the given path. The object is serialized
// as gob, optionally compressed with flate (as gzip) and optionally encrypted with
// AES-GCM. The encryption key must be 32 bytes long. If the file exists, it's
// overwritten, otherwise created.
func persist(filePath string, obj any, compress bool, encryptionKey string) error {
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
	// If path exists and it's a directory, return an error.
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

	// We want to:
	// Encode as gob -> compress with flate -> encrypt with AES-GCM -> write file.
	// To reduce memory usage we chain the writers instead of buffering, so we start
	// from the end. For AES GCM sealing the stdlib doesn't provide a writer though.

	var w io.Writer
	if encryptionKey == "" {
		w = f
	} else {
		w = &bytes.Buffer{}
	}
	if compress {
		gzw := gzip.NewWriter(w)
		defer gzw.Close()
		w = gzw
	}
	enc := gob.NewEncoder(w)

	// Start encoding, it will write to the chain of writers.
	if err := enc.Encode(obj); err != nil {
		return fmt.Errorf("couldn't encode or write object: %w", err)
	}

	// Without encyrption, the chain is done and the file is written.
	if encryptionKey == "" {
		return nil
	}

	// Otherwise, encrypt and then write to the file
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
	// w is a *bytes.Buffer
	buf := w.(*bytes.Buffer)
	encrypted := gcm.Seal(nonce, nonce, buf.Bytes(), nil)
	_, err = f.Write(encrypted)
	if err != nil {
		return fmt.Errorf("couldn't write encrypted data: %w", err)
	}

	return nil
}

// read reads an object from a file at the given path. The object is deserialized
// from gob. `obj` must be a pointer to an instantiated object.
func read(filePath string, obj any) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("couldn't open file '%s': %w", filePath, err)
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	err = dec.Decode(obj)
	if err != nil {
		return fmt.Errorf("couldn't decode or read object: %w", err)
	}

	return nil
}
