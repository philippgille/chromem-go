package chromem

import (
	"errors"
	"math"
)

const isNormalizedPrecisionTolerance = 1e-6

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

// subtractVector subtracts vector b from vector a in place.
func subtractVector(a, b []float32) []float32 {
	res := make([]float32, len(a))

	for i := range a {
		res[i] = a[i] - b[i]
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
