package chromem

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func TestNewEmbeddingFuncOllama(t *testing.T) {
	model := "model-small"
	baseURLSuffix := "/api"
	prompt := "hello world"

	wantBody, err := json.Marshal(map[string]string{
		"model":  model,
		"prompt": prompt,
	})
	if err != nil {
		t.Error("unexpected error:", err)
	}
	wantRes := []float32{-0.1, 0.1, 0.2}

	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check URL
		if !strings.HasSuffix(r.URL.Path, baseURLSuffix+"/embeddings") {
			t.Error("expected URL", baseURLSuffix+"/embeddings", "got", r.URL.Path)
		}
		// Check method
		if r.Method != "POST" {
			t.Error("expected method POST, got", r.Method)
		}
		// Check headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type header", "application/json", "got", r.Header.Get("Content-Type"))
		}
		// Check body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error("unexpected error:", err)
		}
		if !bytes.Equal(body, wantBody) {
			t.Error("expected body", wantBody, "got", body)
		}

		// Write response
		resp := ollamaResponse{
			Embedding: wantRes,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	// Get port from URL
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Error("unexpected error:", err)
	}
	// TODO: It's bad to overwrite a global var for testing. Follow-up with a change
	// to allow passing custom URLs to the function.
	baseURLOllama = strings.Replace(baseURLOllama, "11434", u.Port(), 1)

	f := NewEmbeddingFuncOllama(model)
	res, err := f(context.Background(), prompt)
	if err != nil {
		t.Error("expected nil, got", err)
	}
	if slices.Compare(wantRes, res) != 0 {
		t.Error("expected res", wantRes, "got", res)
	}
}
