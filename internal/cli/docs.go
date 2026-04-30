package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/cache"
	"github.com/kevin-burns/c7search/internal/client"
	"github.com/kevin-burns/c7search/internal/output"
)

const docsCacheTTL = 24 * time.Hour

func newDocsCmd() *cobra.Command {
	var topic string
	var tokens int

	cmd := &cobra.Command{
		Use:   "docs <library-id>",
		Short: "Fetch documentation for a Context7 library ID",
		Long: `Fetch trimmed documentation. Library IDs look like /vercel/next.js or
@types/node — leading slash optional, case-insensitive.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			c := newAPIClient(cmd)
			store := newCacheStore(cmd)

			format := resolveFormat(cmd, output.FormatMarkdown)
			wireFormat := "txt"
			if format == output.FormatJSON {
				wireFormat = "json"
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), client.TimeoutDocs)
			defer cancel()

			doc, err := fetchDocs(ctx, c, store, resolveAPIKey(cmd), id,
				client.DocsOptions{Topic: topic, Tokens: tokens, Format: wireFormat})
			if err != nil {
				return err
			}
			return output.RenderDoc(cmd.OutOrStdout(), doc, format)
		},
	}
	cmd.Flags().StringVar(&topic, "topic", "", "narrow the docs to a topic (e.g. 'routing')")
	cmd.Flags().IntVar(&tokens, "tokens", 5000, "approx token budget for the response")
	return cmd
}

// fetchDocs runs the docs request through the v2 unified context endpoint
// (matches github.com/intellectronica/agent-skills/skills/context7) with a
// per-API-key-scoped cache. The cache key includes a "v2" tag so an old
// disk cache from the v1 era is treated as a clean miss.
func fetchDocs(ctx context.Context, c *client.Client, store *cache.FS, apiKey, id string, opts client.DocsOptions) (client.Doc, error) {
	key := cache.Key("docs-v2",
		apiKeyScope(apiKey),
		client.NormalizeLibraryID(id),
		opts.Topic,
		strconv.Itoa(opts.Tokens),
		opts.Format,
	)
	var cached client.Doc
	if hit, _ := store.Get(key, &cached); hit {
		return cached, nil
	}
	doc, err := c.DocsV2(ctx, id, opts)
	if err != nil {
		return client.Doc{}, fmt.Errorf("fetch docs for %s: %w", client.NormalizeLibraryID(id), err)
	}
	_ = store.Put(key, docsCacheTTL, doc)
	return doc, nil
}
