package chromem

import (
	"errors"
	"math"
)

// cosineSimilarity calculates the cosine similarity between two vectors.
// Vectors are normalized first.
// The resulting value represents the similarity, so a higher value means the
// vectors are more similar.
func cosineSimilarity(a, b []float32) (float32, error) {
	// The vectors must have the same length
	if len(a) != len(b) {
		return 0, errors.New("vectors must have the same length")
	}

	x, y := normalizeVector(a), normalizeVector(b)
	var dotProduct float32
	for i := range x {
		dotProduct += x[i] * y[i]
	}
	// Vectors are already normalized, so no need to divide by magnitudes

	return dotProduct, nil
}

func normalizeVector(v []float32) []float32 {
	var norm float32
	for _, val := range v {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	res := make([]float32, len(v))
	for i, val := range v {
		res[i] = val / norm
	}

	return res
}
