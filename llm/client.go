package llm

import (
	"context"
	"fmt"
	"reader/config"

	einoembed "github.com/cloudwego/eino-ext/components/embedding/openai"
	einomodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// Client wraps Eino ChatModel and Embedder for all LLM operations.
// ChatModel handles text generation; Embedder handles vector embeddings.
type Client struct {
	chatModel *einomodel.ChatModel
	embedder  *einoembed.Embedder
}

// NewClient creates an LLM client backed by Eino's OpenAI-compatible components.
// cfg.BaseURL must point to an OpenAI-compatible API endpoint (including local
// servers like Ollama at http://localhost:11434/v1).
func NewClient(ctx context.Context, cfg config.LLMConfig) (*Client, error) {
	cm, err := einomodel.NewChatModel(ctx, &einomodel.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("create chat model: %w", err)
	}

	emb, err := einoembed.NewEmbedder(ctx, &einoembed.EmbeddingConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.EmbedModel,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("create embedder: %w", err)
	}

	return &Client{
		chatModel: cm,
		embedder:  emb,
	}, nil
}

// Chat sends a system + user message pair and returns the model's text response.
// This is a convenience wrapper around ChatModel.Generate.
func (c *Client) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: userPrompt},
	}

	resp, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}

	return resp.Content, nil
}

// Embed generates embedding vectors for a batch of texts.
// Returns one vector per input text, in the same order. Each vector's
// dimension matches the model (e.g. 1536 for text-embedding-3-small).
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	return c.embedder.EmbedStrings(ctx, texts)
}
