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
		"3": {
			ID:      "3",
			Content: "bonjour and hello foo baz bom",
		},
		"4": {
			ID:      "4",
			Content: "bonjour and hello foo bar baz",
		},
		"5": {
			ID:      "5",
			Content: "bonjour and hello spam eggs",
		},
	}

	tt := []struct {
		name          string
		where         map[string]string
		whereDocument []WhereDocument
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
			whereDocument: []WhereDocument{{Operator: WhereDocumentOperatorContains, Value: "llo"}},
			want:          []*Document{docs["1"], docs["2"], docs["3"], docs["4"], docs["5"]},
		},
		{
			name:          "content contains one",
			where:         nil,
			whereDocument: []WhereDocument{{Operator: WhereDocumentOperatorContains, Value: "hallo"}},
			want:          []*Document{docs["2"]},
		},
		{
			name:          "content contains none",
			where:         nil,
			whereDocument: []WhereDocument{{Operator: WhereDocumentOperatorContains, Value: "salute"}},
			want:          nil,
		},
		{
			name:          "content not_contains all",
			where:         nil,
			whereDocument: []WhereDocument{{Operator: WhereDocumentOperatorNotContains, Value: "bonjour"}},
			want:          []*Document{docs["1"], docs["2"]},
		},
		{
			name:          "content not_contains one",
			where:         nil,
			whereDocument: []WhereDocument{{Operator: WhereDocumentOperatorNotContains, Value: "hello"}},
			want:          []*Document{docs["2"]},
		},
		{
			name:          "meta and content match",
			where:         map[string]string{"language": "de"},
			whereDocument: []WhereDocument{{Operator: WhereDocumentOperatorContains, Value: "hallo"}},
			want:          []*Document{docs["2"]},
		},
		{
			name:          "meta + contains + not_contains",
			where:         map[string]string{"language": "de"},
			whereDocument: []WhereDocument{{Operator: WhereDocumentOperatorContains, Value: "hallo"}, {Operator: WhereDocumentOperatorNotContains, Value: "bonjour"}},
			want:          []*Document{docs["2"]},
		},
		{
			name: "contains or (contains and not_contains",
			whereDocument: []WhereDocument{
				{Operator: WhereDocumentOperatorOr, WhereDocuments: []WhereDocument{
					{Operator: WhereDocumentOperatorContains, Value: "bar"},
					{Operator: WhereDocumentOperatorAnd, WhereDocuments: []WhereDocument{
						{Operator: WhereDocumentOperatorContains, Value: "bonjour"},
						{Operator: WhereDocumentOperatorNotContains, Value: "foo"},
					},
					},
				}},
			},
			want: []*Document{docs["4"], docs["5"]},
		},
	}

	// To avoid issues with checking equality of concurrently produced slices, we sort by ID
	sortDocs := func(d []*Document) {
		slices.SortFunc(d, func(d1, d2 *Document) int {
			if d1.ID < d2.ID {
				return -1
			}
			if d1.ID > d2.ID {
				return 1
			}
			return 0
		})
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := filterDocs(docs, tc.where, tc.whereDocument)
			sortDocs(got)
			sortDocs(tc.want)

			if !reflect.DeepEqual(got, tc.want) {
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
