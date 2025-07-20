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
		"model": model,
		"input": prompt,
	})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	wantRes := [][]float32{{-0.40824828, 0.40824828, 0.81649655}} // normalized version of `{-0.1, 0.1, 0.2}`

	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check URL
		if !strings.HasSuffix(r.URL.Path, baseURLSuffix+"/embed") {
			t.Fatal("expected URL", baseURLSuffix+"/embed", "got", r.URL.Path)
		}
		// Check method
		if r.Method != "POST" {
			t.Fatal("expected method POST, got", r.Method)
		}
		// Check headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatal("expected Content-Type header", "application/json", "got", r.Header.Get("Content-Type"))
		}
		// Check body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if !bytes.Equal(body, wantBody) {
			t.Fatal("expected body", wantBody, "got", body)
		}

		// Write response
		resp := ollamaResponse{
			Embeddings: wantRes,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	// Get port from URL
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	f := NewEmbeddingFuncOllama(model, strings.Replace(defaultBaseURLOllama, "11434", u.Port(), 1))
	res, err := f(context.Background(), prompt)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	if slices.Compare(wantRes[0], res) != 0 {
		t.Fatal("expected res", wantRes, "got", res)
	}
}
