package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/client"
	"github.com/kevin-burns/c7search/internal/output"
)

func newAskCmd() *cobra.Command {
	var topic string
	var tokens int

	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Resolve+fetch in one shot — find the best library and pull its docs",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := strings.Join(args, " ")
			apiKey := resolveAPIKey(cmd)
			c := newAPIClient(cmd)
			store := newCacheStore(cmd)

			ctx, cancel := context.WithTimeout(cmd.Context(), client.TimeoutDocs)
			defer cancel()

			libs, err := resolveLibraries(ctx, c, store, apiKey, question)
			if err != nil {
				return err
			}
			pick, err := pickBest(libs)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "resolved: %s  (trust=%.1f, snippets=%d)\n",
				pick.ID, pick.TrustScore, pick.TotalSnippets)

			format := resolveFormat(cmd, output.FormatMarkdown)
			wireFormat := "txt"
			if format == output.FormatJSON {
				wireFormat = "json"
			}
			effectiveTopic := topic
			if effectiveTopic == "" {
				effectiveTopic = extractTopic(question)
			}

			doc, err := fetchDocs(ctx, c, store, apiKey, pick.ID,
				client.DocsOptions{Topic: effectiveTopic, Tokens: tokens, Format: wireFormat})
			if err != nil {
				return err
			}
			return output.RenderDoc(cmd.OutOrStdout(), doc, format)
		},
	}
	cmd.Flags().StringVar(&topic, "topic", "", "override the auto-derived topic")
	cmd.Flags().IntVar(&tokens, "tokens", 5000, "approx token budget for the response")
	return cmd
}

// pickBest chooses the library with the highest trust score. Ties broken by
// snippet count, then star count.
func pickBest(libs []client.Library) (client.Library, error) {
	if len(libs) == 0 {
		return client.Library{}, errNoResults
	}
	sort.SliceStable(libs, func(i, j int) bool {
		if libs[i].TrustScore != libs[j].TrustScore {
			return libs[i].TrustScore > libs[j].TrustScore
		}
		if libs[i].TotalSnippets != libs[j].TotalSnippets {
			return libs[i].TotalSnippets > libs[j].TotalSnippets
		}
		return libs[i].Stars > libs[j].Stars
	})
	if libs[0].TrustScore == 0 && libs[0].TotalSnippets == 0 {
		return client.Library{}, errors.New("no library with trust or snippet data")
	}
	return libs[0], nil
}

// extractTopic is a simple heuristic that picks meaningful tokens out of a
// natural-language question. Stop-words are dropped; the remainder is
// space-joined and capped. Documented as a heuristic — users can override
// with --topic.
func extractTopic(question string) string {
	stopwords := map[string]bool{
		"how": true, "do": true, "i": true, "to": true, "in": true, "with": true,
		"the": true, "a": true, "an": true, "of": true, "for": true, "is": true,
		"what": true, "are": true, "can": true, "?": true, "and": true,
	}
	fields := strings.Fields(strings.ToLower(question))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimRight(f, "?.,;:")
		if stopwords[f] || len(f) < 2 {
			continue
		}
		out = append(out, f)
	}
	topic := strings.Join(out, " ")
	// Cap on rune boundary so multi-byte UTF-8 isn't sliced mid-codepoint.
	const maxBytes = 80
	if len(topic) > maxBytes {
		i := maxBytes
		for i > 0 && !utf8.RuneStart(topic[i]) {
			i--
		}
		topic = topic[:i]
	}
	return topic
}
