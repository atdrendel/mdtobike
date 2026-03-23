package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for bikemark.

To load completions:

Bash:
  $ source <(bikemark completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ bikemark completion bash > /etc/bash_completion.d/bikemark
  # macOS:
  $ bikemark completion bash > $(brew --prefix)/etc/bash_completion.d/bikemark

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ bikemark completion zsh > "${fpath[1]}/_bikemark"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ bikemark completion fish | source

  # To load completions for each session, execute once:
  $ bikemark completion fish > ~/.config/fish/completions/bikemark.fish

PowerShell:
  PS> bikemark completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> bikemark completion powershell > bikemark.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
