package chromem

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func negativeDimensionCollection(t *testing.T, negative []float32, embedErr error) *Collection {
	t.Helper()
	ctx := context.Background()
	embed := func(_ context.Context, text string) ([]float32, error) {
		if text == "query" {
			return []float32{1, 0, 0}, nil
		}
		return negative, embedErr
	}
	c, err := NewDB().CreateCollection("negative-dimensions", nil, embed)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddDocuments(ctx, []Document{
		{ID: "a", Embedding: []float32{1, 0, 0}},
		{ID: "b", Embedding: []float32{0, 1, 0}},
	}, 1); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestQueryWithOptionsNegativeDimensions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		negative []float32
		fromText bool
	}{
		{name: "short_embedding", negative: []float32{0, 1}},
		{name: "long_embedding", negative: []float32{0, 1, 0, 1}},
		{name: "short_text_embedding", negative: []float32{0, 1}, fromText: true},
		{name: "long_text_embedding", negative: []float32{0, 1, 0, 1}, fromText: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := negativeDimensionCollection(t, tc.negative, nil)
			options := QueryOptions{
				QueryEmbedding: []float32{1, 0, 0},
				NResults:       2,
				Negative: NegativeQueryOptions{
					Embedding: tc.negative,
					Mode:      NEGATIVE_MODE_SUBTRACT,
				},
			}
			if tc.fromText {
				options.QueryEmbedding = nil
				options.QueryText = "query"
				options.Negative.Embedding = nil
				options.Negative.Text = "negative"
			}
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("mismatched embeddings must return an error, not panic: %v", p)
				}
			}()
			results, err := c.QueryWithOptions(context.Background(), options)
			if err == nil {
				t.Fatalf("mismatched embeddings returned successful results: %+v", results)
			}
			if results != nil {
				t.Errorf("failed query returned results: %+v", results)
			}
			doc, err := c.GetByID(context.Background(), "a")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(doc.Embedding, []float32{1, 0, 0}) {
				t.Errorf("failed query changed stored embedding: %v", doc.Embedding)
			}
		})
	}
}

func TestQueryWithOptionsNegativeDimensionControls(t *testing.T) {
	embedErr := errors.New("negative embedding failed")
	for _, tc := range []struct {
		name     string
		negative []float32
		fromText bool
		mode     NegativeMode
		embedErr error
		wantErr  bool
		wantIDs  []string
	}{
		{name: "nil_negative", mode: NEGATIVE_MODE_SUBTRACT, wantIDs: []string{"a", "b"}},
		{name: "empty_negative", negative: []float32{}, mode: NEGATIVE_MODE_SUBTRACT, wantIDs: []string{"a", "b"}},
		{name: "matching_subtract", negative: []float32{0, 1, 0}, mode: NEGATIVE_MODE_SUBTRACT, wantIDs: []string{"a", "b"}},
		{name: "matching_text_subtract", negative: []float32{0, 1, 0}, fromText: true, mode: NEGATIVE_MODE_SUBTRACT, wantIDs: []string{"a", "b"}},
		{name: "matching_filter", negative: []float32{0, 1, 0}, mode: NEGATIVE_MODE_FILTER, wantIDs: []string{"a"}},
		{name: "mismatched_filter", negative: []float32{0, 1}, mode: NEGATIVE_MODE_FILTER, wantErr: true},
		{name: "unsupported_mode", negative: []float32{0, 1}, mode: "invalid", wantErr: true},
		{name: "embedding_error", fromText: true, mode: NEGATIVE_MODE_SUBTRACT, embedErr: embedErr, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := negativeDimensionCollection(t, tc.negative, tc.embedErr)
			options := QueryOptions{
				QueryEmbedding: []float32{1, 0, 0},
				NResults:       2,
				Negative:       NegativeQueryOptions{Embedding: tc.negative, Mode: tc.mode},
			}
			if tc.fromText {
				options.QueryEmbedding = nil
				options.QueryText = "query"
				options.Negative.Embedding = nil
				options.Negative.Text = "negative"
			}
			results, err := c.QueryWithOptions(context.Background(), options)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tc.embedErr != nil && !errors.Is(err, tc.embedErr) {
					t.Errorf("embedding error not preserved: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, len(results))
			for i, result := range results {
				ids[i] = result.ID
			}
			if !reflect.DeepEqual(ids, tc.wantIDs) {
				t.Errorf("result IDs = %v, want %v", ids, tc.wantIDs)
			}
		})
	}
}
