package search

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
)

const DefaultModel = openai.EmbeddingModelTextEmbedding3Small

// EmbeddingClient defines the interface for generating embeddings
type EmbeddingClient interface {
	CreateEmbedding(ctx context.Context, input []string, model openai.EmbeddingModel) ([][]float64, error)
}

// OpenAIClient wraps the OpenAI client to implement EmbeddingClient
type OpenAIClient struct {
	client openai.Client
}

func NewOpenAIClient(client openai.Client) *OpenAIClient {
	return &OpenAIClient{client: client}
}

func (c *OpenAIClient) CreateEmbedding(ctx context.Context, input []string, model openai.EmbeddingModel) ([][]float64, error) {
	resp, err := c.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: input,
		},
		Model: model,
	})
	if err != nil {
		return nil, err
	}

	results := make([][]float64, len(resp.Data))
	for _, emb := range resp.Data {
		results[emb.Index] = emb.Embedding
	}
	return results, nil
}

type Embedder struct {
	client EmbeddingClient
	model  openai.EmbeddingModel
}

func NewEmbedder(client openai.Client, model openai.EmbeddingModel) *Embedder {
	if model == "" {
		model = DefaultModel
	}
	return &Embedder{
		client: NewOpenAIClient(client),
		model:  model,
	}
}

// NewEmbedderWithClient creates an Embedder with a custom EmbeddingClient (useful for testing)
func NewEmbedderWithClient(client EmbeddingClient, model openai.EmbeddingModel) *Embedder {
	if model == "" {
		model = DefaultModel
	}
	return &Embedder{client: client, model: model}
}

func (e *Embedder) Embed(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := e.client.CreateEmbedding(ctx, []string{text}, e.model)
	if err != nil {
		return nil, fmt.Errorf("create embedding: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return embeddings[0], nil
}

func (e *Embedder) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings, err := e.client.CreateEmbedding(ctx, texts, e.model)
	if err != nil {
		return nil, fmt.Errorf("create embeddings: %w", err)
	}

	return embeddings, nil
}

// Model returns the embedding model being used
func (e *Embedder) Model() openai.EmbeddingModel {
	return e.model
}
