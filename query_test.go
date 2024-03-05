package chromem

import (
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
