// Package cli wires the cobra command tree. Subcommands read their
// configuration from the *cobra.Command they're handed (cmd.Flags()) so
// no package-level mutable state survives across Execute() invocations.
// Tests can build a fresh command tree per case and run them in parallel.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/cache"
	"github.com/kevin-burns/c7search/internal/client"
	"github.com/kevin-burns/c7search/internal/output"
	"github.com/kevin-burns/c7search/internal/version"
)

// newRootCmd builds the root command. Each call returns a fresh tree with
// its own flag values — no package-level state.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "c7search",
		Short:         "CLI for the Context7 documentation API",
		Long:          "c7search resolves library IDs and fetches up-to-date documentation from context7.com — same data the Context7 MCP server exposes, but as a single static binary.",
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// Cobra's default version template includes the binary name twice; ours
	// is already self-describing.
	root.SetVersionTemplate("{{.Version}}\n")

	root.PersistentFlags().String("api-key", "", "Context7 API key (overrides $CONTEXT7_API_KEY)")
	root.PersistentFlags().Bool("json", false, "shorthand for --format json")
	root.PersistentFlags().String("format", "", "output format: text | md | json")
	root.PersistentFlags().Bool("no-cache", false, "bypass on-disk cache for this invocation")
	root.PersistentFlags().Bool("debug", false, "verbose stderr logging (secrets redacted)")

	root.AddCommand(
		newResolveCmd(),
		newDocsCmd(),
		newAskCmd(),
		newAuthCmd(),
		newCacheCmd(),
		newVersionCmd(),
	)
	return root
}

// Execute is the entry point invoked from main. Returns the process exit
// code (0 ok, 1 no results, 2 API error, 3 auth error, 4 usage/config).
func Execute() int {
	return run(os.Stdout, os.Stderr, os.Args[1:])
}

// run is the testable form of Execute. It builds a fresh command tree and
// drives it with the given args/writers — no global state touched.
func run(stdout, stderr io.Writer, args []string) int {
	root := newRootCmd()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		// Some commands own their user message (e.g. `auth status` already
		// printed "status: invalid key"). They surface errSilent to keep
		// classify() honest without producing duplicate stderr noise.
		if !errors.Is(err, errSilent) {
			fmt.Fprintln(stderr, "error:", err)
		}
		return classify(err)
	}
	return 0
}

// resolveAPIKey applies the documented precedence: --api-key > env > none.
// Reads the flag from cmd directly so two parallel command trees don't
// share state.
func resolveAPIKey(cmd *cobra.Command) string {
	if k, _ := cmd.Flags().GetString("api-key"); k != "" {
		return k
	}
	return os.Getenv("CONTEXT7_API_KEY")
}

// resolveFormat chooses an output.Format from --json, --format, and a
// per-command default. --json takes precedence over --format.
func resolveFormat(cmd *cobra.Command, defaultFmt output.Format) output.Format {
	if v, _ := cmd.Flags().GetBool("json"); v {
		return output.FormatJSON
	}
	if v, _ := cmd.Flags().GetString("format"); v != "" {
		return output.ParseFormat(v)
	}
	return defaultFmt
}

func noCacheFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("no-cache")
	return v
}

func debugFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("debug")
	return v
}

// newAPIClient is the shared factory; tests can replace it.
var newAPIClient = func(cmd *cobra.Command) *client.Client {
	return client.New(client.Options{
		APIKey: resolveAPIKey(cmd),
		Logger: newDebugLogger(cmd),
	})
}

// newCacheStore is the shared factory; tests can replace it. Returns nil
// when --no-cache is set or when the cache cannot be opened (degrades
// silently — a cache failure must never break a command).
var newCacheStore = func(cmd *cobra.Command) *cache.FS {
	if noCacheFlag(cmd) {
		return nil
	}
	c, err := cache.New()
	if err != nil {
		if debugFlag(cmd) {
			fmt.Fprintln(cmd.ErrOrStderr(), "cache disabled:", err)
		}
		return nil
	}
	return c
}
