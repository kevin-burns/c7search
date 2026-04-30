package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/client"
	"github.com/kevin-burns/c7search/internal/redact"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Inspect API-key configuration and connectivity",
	}
	cmd.AddCommand(newAuthStatusCmd())
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether an API key is loaded and whether the API responds",
		RunE: func(cmd *cobra.Command, args []string) error {
			key := resolveAPIKey(cmd)
			flagSet, _ := cmd.Flags().GetString("api-key")
			out := cmd.OutOrStdout()
			source := "anonymous (no key)"
			switch {
			case flagSet != "":
				source = "--api-key flag"
			case key != "":
				source = "$CONTEXT7_API_KEY"
			}

			_, _ = fmt.Fprintf(out, "key:    %s\n", redact.RedactAPIKey(key))
			_, _ = fmt.Fprintf(out, "source: %s\n", source)

			c := newAPIClient(cmd)
			ctx, cancel := context.WithTimeout(cmd.Context(), client.TimeoutAuth)
			defer cancel()

			if err := c.Verify(ctx); err != nil {
				if errors.Is(err, client.ErrUnauthorized) {
					fmt.Fprintf(out, "status: invalid key\n")
					return silentWrap(err)
				}
				fmt.Fprintf(out, "status: API unreachable (%v)\n", err)
				return silentWrap(err)
			}
			fmt.Fprintf(out, "status: ok\n")
			return nil
		},
	}
}
