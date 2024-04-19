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

	var gzw *gzip.Writer
	var enc *gob.Encoder
	if compress {
		gzw = gzip.NewWriter(w)
		enc = gob.NewEncoder(gzw)
	} else {
		enc = gob.NewEncoder(w)
	}

	// Start encoding, it will write to the chain of writers.
	if err := enc.Encode(obj); err != nil {
		return fmt.Errorf("couldn't encode or write object: %w", err)
	}

	// If compressing, close the gzip writer. Otherwise the gzip footer won't be
	// written yet. When using encryption (and w is a buffer) then we'll encrypt
	// an incomplete file. Without encryption when we return here and having
	// a deferred Close(), there might be a silenced error.
	if compress {
		err = gzw.Close()
		if err != nil {
			return fmt.Errorf("couldn't close gzip writer: %w", err)
		}
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
// from gob. `obj` must be a pointer to an instantiated object. The file may
// optionally be compressed as gzip and/or encrypted with AES-GCM. The encryption
// key must be 32 bytes long.
func read(filePath string, obj any, encryptionKey string) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}
	// AES 256 requires a 32 byte key
	if encryptionKey != "" {
		if len(encryptionKey) != 32 {
			return errors.New("encryption key must be 32 bytes long")
		}
	}

	// We want to:
	// Read file -> decrypt with AES-GCM -> decompress with flate -> decode as gob
	// To reduce memory usage we chain the readers instead of buffering, so we start
	// from the end. For the decryption there's no reader though.

	var r io.Reader

	// Decrypt if an encryption key is provided
	if encryptionKey != "" {
		encrypted, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("couldn't read file: %w", err)
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

		r = bytes.NewReader(data)
	} else {
		var err error
		r, err = os.Open(filePath)
		if err != nil {
			return fmt.Errorf("couldn't open file: %w", err)
		}
	}

	// Determine if the file is compressed
	magicNumber := make([]byte, 2)
	_, err := r.Read(magicNumber)
	if err != nil {
		return fmt.Errorf("couldn't read magic number to determine whether the file is compressed: %w", err)
	}
	var compressed bool
	if magicNumber[0] == 0x1f && magicNumber[1] == 0x8b {
		compressed = true
	}

	// Reset reader. Both file and bytes.Reader support seeking.
	if s, ok := r.(io.Seeker); !ok {
		return fmt.Errorf("reader doesn't support seeking")
	} else {
		_, err := s.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("couldn't reset reader: %w", err)
		}
	}

	if compressed {
		gzr, err := gzip.NewReader(r)
		if err != nil {
			return fmt.Errorf("couldn't create gzip reader: %w", err)
		}
		defer gzr.Close()
		r = gzr
	}

	dec := gob.NewDecoder(r)
	err = dec.Decode(obj)
	if err != nil {
		return fmt.Errorf("couldn't decode object: %w", err)
	}

	return nil
}

// remove removes a file at the given path. If the file doesn't exist, it's a no-op.
func remove(filePath string) error {
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
