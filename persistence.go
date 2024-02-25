package chromem

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
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
// as gob.
func persist(filePath string, obj any) error {
	filePath += ".gob"

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("couldn't create file '%s': %w", filePath, err)
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	err = enc.Encode(obj)
	if err != nil {
		return fmt.Errorf("couldn't encode or write object: %w", err)
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
