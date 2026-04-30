package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kevin-burns/c7search/internal/cache"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the on-disk response cache",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Remove every cached search/docs entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cache.New()
			if err != nil {
				return err
			}
			n, err := c.Clear()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %d cached entries from %s\n", n, c.Dir)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the cache directory path",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cache.New()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), c.Dir)
			return nil
		},
	})
	return cmd
}
