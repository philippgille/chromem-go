package chromem_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/philippgille/chromem-go"
)

type openAIResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func TestNewEmbeddingFuncOpenAICompat(t *testing.T) {
	apiKey := "secret"
	model := "model-small"
	baseURLSuffix := "/v1"
	input := "hello world"

	wantBody, err := json.Marshal(map[string]string{
		"input": input,
		"model": model,
	})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	wantRes := []float32{-0.1, 0.1, 0.2}

	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check URL
		if !strings.HasSuffix(r.URL.Path, baseURLSuffix+"/embeddings") {
			t.Fatal("expected URL", baseURLSuffix+"/embeddings", "got", r.URL.Path)
		}
		// Check method
		if r.Method != "POST" {
			t.Fatal("expected method POST, got", r.Method)
		}
		// Check headers
		if r.Header.Get("Authorization") != "Bearer "+apiKey {
			t.Fatal("expected Authorization header", "Bearer "+apiKey, "got", r.Header.Get("Authorization"))
		}
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
		resp := openAIResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
			}{
				{Embedding: wantRes},
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()
	baseURL := ts.URL + baseURLSuffix

	f := chromem.NewEmbeddingFuncOpenAICompat(baseURL, apiKey, model)
	res, err := f(context.Background(), input)
	if err != nil {
		t.Fatal("expected nil, got", err)
	}
	if slices.Compare(wantRes, res) != 0 {
		t.Fatal("expected res", wantRes, "got", res)
	}
}
