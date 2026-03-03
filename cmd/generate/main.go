package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brojonat/vssss/internal/db"
	"github.com/brojonat/vssss/internal/search"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/urfave/cli/v3"
)

type VSSNode struct {
	Description string              `json:"description"`
	Type        string              `json:"type"`
	DataType    string              `json:"datatype"`
	Unit        string              `json:"unit"`
	Children    map[string]*VSSNode `json:"children"`
}

type signal struct {
	Path        string
	Description string
	Type        string
	DataType    string
	Unit        string
}

func main() {
	cmd := &cli.Command{
		Name:  "generate",
		Usage: "Generate embeddings database from VSS JSON",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "vss",
				Aliases: []string{"v"},
				Value:   "vss.json",
				Usage:   "path to vss.json",
			},
			&cli.StringFlag{
				Name:    "db",
				Aliases: []string{"d"},
				Value:   "signals.db",
				Usage:   "output database path",
			},
			&cli.IntFlag{
				Name:    "batch",
				Aliases: []string{"b"},
				Value:   100,
				Usage:   "embedding batch size",
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	vssFile := cmd.String("vss")
	dbFile := cmd.String("db")
	batchSize := int(cmd.Int("batch"))

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable required")
	}

	// Read VSS JSON
	data, err := os.ReadFile(vssFile)
	if err != nil {
		return fmt.Errorf("read vss.json: %w", err)
	}

	var root map[string]*VSSNode
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parse vss.json: %w", err)
	}

	// Extract all signals
	var signals []signal
	for name, node := range root {
		extractSignals(name, node, &signals)
	}

	log.Printf("Extracted %d signals from VSS", len(signals))

	// Filter to only signals with descriptions
	var withDesc []signal
	for _, s := range signals {
		if s.Description != "" {
			withDesc = append(withDesc, s)
		}
	}
	log.Printf("Found %d signals with descriptions", len(withDesc))

	// Initialize database
	store, err := db.Open(dbFile)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		return fmt.Errorf("init db: %w", err)
	}

	// Initialize embedder
	var opts []option.RequestOption
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := openai.NewClient(opts...)
	embedder := search.NewEmbedder(client, "")

	// Generate embeddings in batches
	for i := 0; i < len(withDesc); i += batchSize {
		end := i + batchSize
		if end > len(withDesc) {
			end = len(withDesc)
		}
		batch := withDesc[i:end]

		// Build texts for embedding
		texts := make([]string, len(batch))
		for j, s := range batch {
			texts[j] = fmt.Sprintf("%s: %s", s.Path, s.Description)
		}

		log.Printf("Embedding batch %d-%d of %d...", i+1, end, len(withDesc))

		embeddings, err := embedder.EmbedBatch(ctx, texts)
		if err != nil {
			return fmt.Errorf("embed batch: %w", err)
		}

		// Store in database
		for j, emb := range embeddings {
			s := batch[j]
			if err := store.InsertSignal(ctx, s.Path, s.Description, s.Type, s.DataType, s.Unit, emb); err != nil {
				return fmt.Errorf("insert signal: %w", err)
			}
		}

		// Rate limiting
		if end < len(withDesc) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Printf("Successfully generated embeddings database: %s", dbFile)
	return nil
}

func extractSignals(path string, node *VSSNode, signals *[]signal) {
	if node == nil {
		return
	}

	*signals = append(*signals, signal{
		Path:        path,
		Description: node.Description,
		Type:        node.Type,
		DataType:    node.DataType,
		Unit:        node.Unit,
	})

	for name, child := range node.Children {
		extractSignals(path+"."+name, child, signals)
	}
}
