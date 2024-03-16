package chromem

import (
	"errors"
	"fmt"
	"math"
)

const isNormalizedPrecisionTolerance = 1e-6

// cosineSimilarity calculates the cosine similarity between two vectors.
// Vectors are normalized first.
// The resulting value represents the similarity, so a higher value means the
// vectors are more similar.
func cosineSimilarity(a, b []float32) (float32, error) {
	// The vectors must have the same length
	if len(a) != len(b) {
		return 0, errors.New("vectors must have the same length")
	}

	if !isNormalized(a) || !isNormalized(b) {
		a, b = normalizeVector(a), normalizeVector(b)
	}
	dotProduct, err := dotProduct(a, b)
	if err != nil {
		return 0, fmt.Errorf("couldn't calculate dot product: %w", err)
	}

	// Vectors are already normalized, so no need to divide by magnitudes

	return dotProduct, nil
}

// dotProduct calculates the dot product between two vectors.
// It's the same as cosine similarity for normalized vectors.
// The resulting value represents the similarity, so a higher value means the
// vectors are more similar.
func dotProduct(a, b []float32) (float32, error) {
	// The vectors must have the same length
	if len(a) != len(b) {
		return 0, errors.New("vectors must have the same length")
	}

	var dotProduct float32
	for i := range a {
		dotProduct += a[i] * b[i]
	}

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

// isNormalized checks if the vector is normalized.
func isNormalized(v []float32) bool {
	var sqSum float64
	for _, val := range v {
		sqSum += float64(val) * float64(val)
	}
	magnitude := math.Sqrt(sqSum)
	return math.Abs(magnitude-1) < isNormalizedPrecisionTolerance
}
