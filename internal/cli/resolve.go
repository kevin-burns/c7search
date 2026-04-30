package cli

import (
	"context"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/cache"
	"github.com/kevin-burns/c7search/internal/client"
	"github.com/kevin-burns/c7search/internal/output"
)

const resolveCacheTTL = 6 * time.Hour

func newResolveCmd() *cobra.Command {
	var (
		limit       int
		libraryName string
	)
	cmd := &cobra.Command{
		Use:   "resolve [query]",
		Short: "Find Context7 library IDs matching a query",
		Long: `Find Context7 library IDs matching a query.

By default, the entire argument is sent as a free-form query to the v1
search endpoint. Pass --library-name to route through the v2 endpoint
which separates the library name from the relevance query — useful when
you know which library you want and just need topic-level ranking.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			c := newAPIClient(cmd)
			store := newCacheStore(cmd)

			ctx, cancel := context.WithTimeout(cmd.Context(), client.TimeoutResolve)
			defer cancel()

			var (
				libs []client.Library
				err  error
			)
			if libraryName != "" {
				libs, err = resolveLibrariesV2(ctx, c, store, resolveAPIKey(cmd), libraryName, query)
			} else {
				libs, err = resolveLibraries(ctx, c, store, resolveAPIKey(cmd), query)
			}
			if err != nil {
				return err
			}
			if len(libs) == 0 {
				return errNoResults
			}
			return output.RenderLibraries(cmd.OutOrStdout(), libs, resolveFormat(cmd, output.FormatText), limit)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "max results to display (0 = unlimited)")
	cmd.Flags().StringVar(&libraryName, "library-name", "",
		"library name (routes through /api/v2/libs/search; query becomes the relevance topic)")
	return cmd
}

// resolveLibraries runs a Search through the cache. The cache key namespaces
// on the API-key identity so anonymous and keyed callers don't poison each
// other's cache; the query is whitespace-collapsed so "next  js" hits the
// same slot as "next js".
func resolveLibraries(ctx context.Context, c *client.Client, store *cache.FS, apiKey, query string) ([]client.Library, error) {
	key := cache.Key("search", apiKeyScope(apiKey), normalizeQuery(query))
	var cached []client.Library
	if hit, _ := store.Get(key, &cached); hit {
		return cached, nil
	}
	libs, err := c.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	_ = store.Put(key, resolveCacheTTL, libs) // cache failure is non-fatal
	return libs, nil
}

// normalizeQuery lowers and collapses runs of whitespace.
func normalizeQuery(q string) string {
	return strings.Join(strings.Fields(strings.ToLower(q)), " ")
}

// resolveLibrariesV2 calls the v2 search endpoint. Cache namespacing
// includes the libraryName so it doesn't collide with v1 free-form keys.
func resolveLibrariesV2(ctx context.Context, c *client.Client, store *cache.FS, apiKey, libraryName, query string) ([]client.Library, error) {
	key := cache.Key("search-v2", apiKeyScope(apiKey), normalizeQuery(libraryName), normalizeQuery(query))
	var cached []client.Library
	if hit, _ := store.Get(key, &cached); hit {
		return cached, nil
	}
	libs, err := c.SearchV2(ctx, libraryName, query)
	if err != nil {
		return nil, err
	}
	_ = store.Put(key, resolveCacheTTL, libs)
	return libs, nil
}
