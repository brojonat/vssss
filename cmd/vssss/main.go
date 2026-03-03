package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/brojonat/vssss"
	"github.com/brojonat/vssss/internal/db"
	"github.com/brojonat/vssss/internal/search"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:      "vssss",
		Usage:     "Semantic search for Vehicle Signal Specification",
		ArgsUsage: "<query>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "num",
				Aliases: []string{"n"},
				Value:   10,
				Usage:   "number of results to return",
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() == 0 {
		return fmt.Errorf("query required\n\nExamples:\n  vssss \"engine temperature\"\n  vssss -n 5 \"battery state of charge\"")
	}

	query := strings.Join(cmd.Args().Slice(), " ")
	limit := int(cmd.Int("num"))

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable required")
	}

	// Open embedded database from memory
	store, err := db.OpenMem("signals.db", vssss.SignalsDB)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()

	// Generate query embedding
	var opts []option.RequestOption
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := openai.NewClient(opts...)
	embedder := search.NewEmbedder(client, "")

	queryEmb, err := embedder.Embed(ctx, query)
	if err != nil {
		return fmt.Errorf("embed query: %w", err)
	}

	// Search
	results, err := store.Search(ctx, queryEmb, limit)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	// Output results as JSON
	type result struct {
		Signal      string `json:"signal"`
		Description string `json:"description"`
	}
	output := make([]result, len(results))
	for i, r := range results {
		output[i] = result{
			Signal:      r.Signal.Path,
			Description: r.Signal.Description,
		}
	}
	return json.NewEncoder(os.Stdout).Encode(output)
}
