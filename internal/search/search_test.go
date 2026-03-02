package search

import (
	"context"
	"errors"
	"testing"

	"github.com/openai/openai-go"
)

type MockEmbeddingClient struct {
	embeddings [][]float64
	err        error
	calls      []mockCall
}

type mockCall struct {
	input []string
	model openai.EmbeddingModel
}

func (m *MockEmbeddingClient) CreateEmbedding(ctx context.Context, input []string, model openai.EmbeddingModel) ([][]float64, error) {
	m.calls = append(m.calls, mockCall{input: input, model: model})
	if m.err != nil {
		return nil, m.err
	}
	return m.embeddings, nil
}

func TestNewEmbedderWithClient(t *testing.T) {
	mock := &MockEmbeddingClient{}

	embedder := NewEmbedderWithClient(mock, "")
	if embedder.Model() != DefaultModel {
		t.Errorf("expected default model %s, got %s", DefaultModel, embedder.Model())
	}

	customModel := openai.EmbeddingModelTextEmbedding3Large
	embedder = NewEmbedderWithClient(mock, customModel)
	if embedder.Model() != customModel {
		t.Errorf("expected model %s, got %s", customModel, embedder.Model())
	}
}

func TestEmbed(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedEmbedding := []float64{0.1, 0.2, 0.3}
		mock := &MockEmbeddingClient{embeddings: [][]float64{expectedEmbedding}}
		embedder := NewEmbedderWithClient(mock, "")

		result, err := embedder.Embed(ctx, "test text")
		if err != nil {
			t.Fatalf("Embed failed: %v", err)
		}

		if len(result) != len(expectedEmbedding) {
			t.Fatalf("expected %d dimensions, got %d", len(expectedEmbedding), len(result))
		}
	})

	t.Run("empty response", func(t *testing.T) {
		mock := &MockEmbeddingClient{embeddings: [][]float64{}}
		embedder := NewEmbedderWithClient(mock, "")

		_, err := embedder.Embed(ctx, "test text")
		if err == nil {
			t.Fatal("expected error for empty response, got nil")
		}
	})

	t.Run("api error", func(t *testing.T) {
		mock := &MockEmbeddingClient{err: errors.New("API error")}
		embedder := NewEmbedderWithClient(mock, "")

		_, err := embedder.Embed(ctx, "test text")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestEmbedBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedEmbeddings := [][]float64{{0.1, 0.2}, {0.3, 0.4}, {0.5, 0.6}}
		mock := &MockEmbeddingClient{embeddings: expectedEmbeddings}
		embedder := NewEmbedderWithClient(mock, "")

		results, err := embedder.EmbedBatch(ctx, []string{"a", "b", "c"})
		if err != nil {
			t.Fatalf("EmbedBatch failed: %v", err)
		}

		if len(results) != len(expectedEmbeddings) {
			t.Fatalf("expected %d embeddings, got %d", len(expectedEmbeddings), len(results))
		}
	})

	t.Run("api error", func(t *testing.T) {
		mock := &MockEmbeddingClient{err: errors.New("rate limit")}
		embedder := NewEmbedderWithClient(mock, "")

		_, err := embedder.EmbedBatch(ctx, []string{"a", "b"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
