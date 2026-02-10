package chromem

import (
	"context"
	"reflect"
	"slices"
	"testing"
)

func TestFilterDocs(t *testing.T) {
	docs := map[string]*Document{
		"1": {
			ID: "1",
			Metadata: map[string]string{
				"language": "en",
			},
			Embedding: []float32{0.1, 0.2, 0.3},
			Content:   "hello world",
		},
		"2": {
			ID: "2",
			Metadata: map[string]string{
				"language": "de",
			},
			Embedding: []float32{0.2, 0.3, 0.4},
			Content:   "hallo welt",
		},
	}

	tt := []struct {
		name          string
		where         map[string]string
		whereDocument map[string]string
		want          []*Document
	}{
		{
			name:          "meta match",
			where:         map[string]string{"language": "de"},
			whereDocument: nil,
			want:          []*Document{docs["2"]},
		},
		{
			name:          "meta no match",
			where:         map[string]string{"language": "fr"},
			whereDocument: nil,
			want:          nil,
		},
		{
			name:          "content contains all",
			where:         nil,
			whereDocument: map[string]string{"$contains": "llo"},
			want:          []*Document{docs["1"], docs["2"]},
		},
		{
			name:          "content contains one",
			where:         nil,
			whereDocument: map[string]string{"$contains": "hallo"},
			want:          []*Document{docs["2"]},
		},
		{
			name:          "content contains none",
			where:         nil,
			whereDocument: map[string]string{"$contains": "bonjour"},
			want:          nil,
		},
		{
			name:          "content not_contains all",
			where:         nil,
			whereDocument: map[string]string{"$not_contains": "bonjour"},
			want:          []*Document{docs["1"], docs["2"]},
		},
		{
			name:          "content not_contains one",
			where:         nil,
			whereDocument: map[string]string{"$not_contains": "hello"},
			want:          []*Document{docs["2"]},
		},
		{
			name:          "meta and content match",
			where:         map[string]string{"language": "de"},
			whereDocument: map[string]string{"$contains": "hallo"},
			want:          []*Document{docs["2"]},
		},
		{
			name:          "meta + contains + not_contains",
			where:         map[string]string{"language": "de"},
			whereDocument: map[string]string{"$contains": "hallo", "$not_contains": "bonjour"},
			want:          []*Document{docs["2"]},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := filterDocs(docs, tc.where, tc.whereDocument)

			if !reflect.DeepEqual(got, tc.want) {
				// If len is 2, the order might be different (function under test
				// is concurrent and order is not guaranteed).
				if len(got) == 2 && len(tc.want) == 2 {
					slices.Reverse(got)
					if reflect.DeepEqual(got, tc.want) {
						return
					}
				}
				t.Fatalf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestNegative(t *testing.T) {
	ctx := context.Background()
	db := NewDB()

	c, err := db.CreateCollection("test", nil, nil)
	if err != nil {
		panic(err)
	}

	if err := c.AddDocuments(ctx, []Document{
		{
			ID:        "1",
			Embedding: testEmbeddings["search_document: Village Builder Game"],
		},
		{
			ID:        "2",
			Embedding: testEmbeddings["search_document: Town Craft Idle Game"],
		},
		{
			ID:        "3",
			Embedding: testEmbeddings["search_document: Some Idle Game"],
		},
	}, 1); err != nil {
		t.Fatalf("failed to add documents: %v", err)
	}

	t.Run("NEGATIVE_MODE_SUBTRACT", func(t *testing.T) {
		res, err := c.QueryWithOptions(ctx, QueryOptions{
			QueryEmbedding: testEmbeddings["search_query: town"],
			NResults:       c.Count(),
			Negative: NegativeQueryOptions{
				Embedding: testEmbeddings["search_query: idle"],
				Mode:      NEGATIVE_MODE_SUBTRACT,
			},
		})
		if err != nil {
			panic(err)
		}

		for _, r := range res {
			t.Logf("%s: %v", r.ID, r.Similarity)
		}

		if len(res) != 3 {
			t.Fatalf("expected 3 results, got %d", len(res))
		}

		// Village Builder Game
		if res[0].ID != "1" {
			t.Fatalf("expected document with ID 1, got %s", res[0].ID)
		}
		// Town Craft Idle Game
		if res[1].ID != "2" {
			t.Fatalf("expected document with ID 2, got %s", res[1].ID)
		}
		// Some Idle Game
		if res[2].ID != "3" {
			t.Fatalf("expected document with ID 3, got %s", res[2].ID)
		}
	})

	t.Run("NEGATIVE_MODE_FILTER", func(t *testing.T) {
		res, err := c.QueryWithOptions(ctx, QueryOptions{
			QueryEmbedding: testEmbeddings["search_query: town"],
			NResults:       c.Count(),
			Negative: NegativeQueryOptions{
				Embedding: testEmbeddings["search_query: idle"],
				Mode:      NEGATIVE_MODE_FILTER,
			},
		})
		if err != nil {
			panic(err)
		}

		for _, r := range res {
			t.Logf("%s: %v", r.ID, r.Similarity)
		}

		if len(res) != 1 {
			t.Fatalf("expected 1 result, got %d", len(res))
		}

		// Village Builder Game
		if res[0].ID != "1" {
			t.Fatalf("expected document with ID 1, got %s", res[0].ID)
		}
	})
}

func TestThreshold(t *testing.T) {
	ctx := context.Background()
	db := NewDB()

	c, err := db.CreateCollection("test-threshold", nil, nil)
	if err != nil {
		panic(err)
	}

	// Add test documents with known embeddings
	if err := c.AddDocuments(ctx, []Document{
		{
			ID:        "high-similarity",
			Embedding: []float32{0.9, 0.1, 0.1}, // Similar to query
		},
		{
			ID:        "medium-similarity",
			Embedding: []float32{0.5, 0.5, 0.5}, // Medium similarity
		},
		{
			ID:        "low-similarity",
			Embedding: []float32{0.1, 0.9, 0.1}, // Low similarity
		},
	}, 1); err != nil {
		t.Fatalf("failed to add documents: %v", err)
	}

	query := []float32{0.8, 0.2, 0.2}

	t.Run("threshold filters low similarity", func(t *testing.T) {
		res, err := c.QueryWithOptions(ctx, QueryOptions{
			QueryEmbedding: query,
			NResults:       c.Count(), // Use actual count
			Threshold:      0.85,      // Only return documents with >= 0.85 similarity
		})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}

		// Should only return high-similarity document
		if len(res) != 1 {
			t.Fatalf("expected 1 result with threshold 0.85, got %d", len(res))
		}

		if res[0].ID != "high-similarity" {
			t.Errorf("expected high-similarity document, got %s", res[0].ID)
		}
	})

	t.Run("threshold 0 returns all results", func(t *testing.T) {
		res, err := c.QueryWithOptions(ctx, QueryOptions{
			QueryEmbedding: query,
			NResults:       c.Count(), // Use actual count
			Threshold:      0,         // No threshold
		})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}

		// Should return all 3 documents
		if len(res) != 3 {
			t.Fatalf("expected 3 results with threshold 0, got %d", len(res))
		}
	})

	t.Run("high threshold returns no results", func(t *testing.T) {
		res, err := c.QueryWithOptions(ctx, QueryOptions{
			QueryEmbedding: query,
			NResults:       c.Count(), // Use actual count
			Threshold:      0.99,      // Very high threshold (higher than 0.98)
		})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}

		// Should return no documents
		if len(res) != 0 {
			t.Errorf("expected 0 results with threshold 0.99, got %d", len(res))
		}
	})

	t.Run("threshold with nResults limit", func(t *testing.T) {
		// Add more documents to test nResults limit
		if err := c.AddDocuments(ctx, []Document{
			{
				ID:        "another-high",
				Embedding: []float32{0.85, 0.15, 0.15},
			},
			{
				ID:        "another-medium",
				Embedding: []float32{0.45, 0.45, 0.45},
			},
		}, 1); err != nil {
			t.Fatalf("failed to add more documents: %v", err)
		}

		res, err := c.QueryWithOptions(ctx, QueryOptions{
			QueryEmbedding: query,
			NResults:       2,   // Only return top 2
			Threshold:      0.4, // But only those with >= 0.4 similarity
		})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}

		// Should return at most 2 results, but both must meet threshold
		if len(res) > 2 {
			t.Errorf("expected at most 2 results, got %d", len(res))
		}

		// Verify all results meet threshold
		for _, r := range res {
			if r.Similarity < 0.4 {
				t.Errorf("result %s has similarity %v below threshold 0.4", r.ID, r.Similarity)
			}
		}
	})

	t.Run("invalid threshold returns error", func(t *testing.T) {
		_, err := c.QueryWithOptions(ctx, QueryOptions{
			QueryEmbedding: query,
			NResults:       1,
			Threshold:      -0.5, // Invalid threshold
		})
		if err == nil {
			t.Error("expected error for negative threshold")
		}

		_, err = c.QueryWithOptions(ctx, QueryOptions{
			QueryEmbedding: query,
			NResults:       1,
			Threshold:      1.5, // Invalid threshold
		})
		if err == nil {
			t.Error("expected error for threshold > 1")
		}
	})
}
