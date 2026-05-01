package cmd

import (
	"fmt"

	"github.com/atdrendel/bikemark/internal/version"
	"github.com/spf13/cobra"
)

var fullVersion bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Display the version of bikemark, optionally with build details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if fullVersion {
			fmt.Fprintln(cmd.OutOrStdout(), version.Full())
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), version.Info())
		}
	},
}

func init() {
	versionCmd.Flags().BoolVar(&fullVersion, "full", false, "show full version info including commit and build date")
	rootCmd.AddCommand(versionCmd)
}
