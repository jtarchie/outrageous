package vector

import (
	"encoding/json"
	"fmt"
	"math"
)

type Vector []float32

func (v *Vector) Scan(value interface{}) error {
	return json.Unmarshal([]byte(value.(string)), v)
}

func (a Vector) cosineSimilarity(b Vector) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must be of the same length")
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i]) // A · B
		normA += float64(a[i]) * float64(a[i])      // ||A||²
		normB += float64(b[i]) * float64(b[i])      // ||B||²
	}

	// Normalize
	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("one of the vectors is zero")
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))), nil
}
