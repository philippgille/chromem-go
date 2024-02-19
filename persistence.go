package chromem

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
)

func hash2hex(name string) string {
	hash := sha256.Sum256([]byte(name))
	return hex.EncodeToString(hash[:])
}

// persist persists an object to a file at the given path.
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
