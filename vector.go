package chromem

import (
	"errors"
	"math"
)

const isNormalizedPrecisionTolerance = 1e-6

var (
	falseVal = false
	trueVal  = true
)

// cosineSimilarity calculates the cosine similarity between two vectors.
// Pass isNormalized=true if the vectors are already normalized, false
// to normalize them, and nil to autodetect.
// The resulting value represents the similarity, so a higher value means the
// vectors are more similar.
func cosineSimilarity(a, b []float32, isNormalized *bool) (float32, error) {
	// The vectors must have the same length
	if len(a) != len(b) {
		return 0, errors.New("vectors must have the same length")
	}

	if isNormalized == nil {
		if !checkNormalized(a) || !checkNormalized(b) {
			isNormalized = &falseVal
		} else {
			isNormalized = &trueVal
		}
	}
	if !*isNormalized {
		a, b = normalizeVector(a), normalizeVector(b)
	}

	var dotProduct float32
	for i := range a {
		dotProduct += a[i] * b[i]
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

// checkNormalized checks if the vector is normalized.
func checkNormalized(v []float32) bool {
	var sqSum float64
	for _, val := range v {
		sqSum += float64(val) * float64(val)
	}
	magnitude := math.Sqrt(sqSum)
	return math.Abs(magnitude-1) < isNormalizedPrecisionTolerance
}
