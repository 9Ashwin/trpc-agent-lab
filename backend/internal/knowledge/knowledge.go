// Package knowledge wires the RAG knowledge base (optional). It is disabled by
// default because it needs an embedding endpoint; enable it via config.
package knowledge

import (
	"context"
	"fmt"

	kbase "trpc.group/trpc-go/trpc-agent-go/knowledge"
	openaiembedder "trpc.group/trpc-go/trpc-agent-go/knowledge/embedder/openai"
	"trpc.group/trpc-go/trpc-agent-go/knowledge/source"
	dirsource "trpc.group/trpc-go/trpc-agent-go/knowledge/source/dir"
	"trpc.group/trpc-go/trpc-agent-go/knowledge/vectorstore/inmemory"

	"github.com/9Ashwin/trpc-agent-lab/backend/internal/config"
)

// Build constructs and loads the RAG knowledge base when enabled. It returns
// (nil, nil) when knowledge is disabled (no embedder configured).
func Build(cfg *config.Config) (kbase.Knowledge, error) {
	if !cfg.KnowledgeEnabled || cfg.EmbedderModel == "" {
		return nil, nil
	}

	emb := openaiembedder.New(
		openaiembedder.WithModel(cfg.EmbedderModel),
		openaiembedder.WithAPIKey(cfg.EmbedderAPIKey),
		openaiembedder.WithBaseURL(cfg.EmbedderBaseURL),
	)

	kb := kbase.New(
		kbase.WithEmbedder(emb),
		kbase.WithVectorStore(inmemory.New()),
		kbase.WithSources([]source.Source{dirsource.New([]string{cfg.KnowledgeDir})}),
	)

	if err := kb.Load(context.Background()); err != nil {
		return nil, fmt.Errorf("load knowledge: %w", err)
	}
	return kb, nil
}
