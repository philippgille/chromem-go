package chromem

import (
	"context"
	"maps"
	"testing"
)

func TestCollection_AddDocumentCopiesMetadata(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		name := "memory"
		if persistent {
			name = "persistent"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			db := NewDB()
			var dir string
			if persistent {
				dir = t.TempDir()
				var err error
				db, err = NewPersistentDB(dir, false)
				if err != nil {
					t.Fatal(err)
				}
			}
			c, err := db.CreateCollection("docs", nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			metadata := map[string]string{"source": "original", "title": "first"}
			want := maps.Clone(metadata)
			if err := c.AddDocument(ctx, Document{
				ID: "1", Embedding: []float32{1, 0}, Metadata: metadata,
			}); err != nil {
				t.Fatal(err)
			}

			// Reusing the caller's metadata map must not change an indexed document.
			metadata["source"] = "other"
			delete(metadata, "title")
			metadata["extra"] = "new"

			doc, err := c.GetByID(ctx, "1")
			if err != nil {
				t.Fatal(err)
			}
			if !maps.Equal(doc.Metadata, want) {
				t.Errorf("stored metadata = %v, want %v", doc.Metadata, want)
			}
			for source, wantCount := range map[string]int{"original": 1, "other": 0} {
				results, err := c.QueryEmbedding(ctx, []float32{1, 0}, 1, map[string]string{"source": source}, nil)
				if err != nil {
					t.Fatal(err)
				}
				if len(results) != wantCount {
					t.Errorf("query source %q returned %d documents, want %d", source, len(results), wantCount)
				}
			}
			if persistent {
				reopened, err := NewPersistentDB(dir, false)
				if err != nil {
					t.Fatal(err)
				}
				reloaded, err := reopened.GetCollection("docs", nil).GetByID(ctx, "1")
				if err != nil {
					t.Fatal(err)
				}
				if !maps.Equal(doc.Metadata, reloaded.Metadata) {
					t.Errorf("in-memory metadata = %v, persisted metadata = %v", doc.Metadata, reloaded.Metadata)
				}
			}
		})
	}
}

func TestCollection_AddDocumentNilMetadata(t *testing.T) {
	ctx := context.Background()
	c, err := NewDB().CreateCollection("docs", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddDocument(ctx, Document{ID: "1", Embedding: []float32{1, 0}}); err != nil {
		t.Fatal(err)
	}
	doc, err := c.GetByID(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Metadata != nil {
		t.Errorf("metadata = %v, want nil", doc.Metadata)
	}
}
