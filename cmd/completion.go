package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/adnanex/dbctl/pkg/ui"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for dbctl.

To load completions:

Bash:
  $ source <(dbctl completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ dbctl completion bash > /etc/bash_completion.d/dbctl
  # macOS:
  $ dbctl completion bash > $(brew --prefix)/etc/bash_completion.d/dbctl

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ dbctl completion zsh > "${fpath[1]}/_dbctl"

Fish:
  $ dbctl completion fish | source

  # To load completions for each session, execute once:
  $ dbctl completion fish > ~/.config/fish/completions/dbctl.fish

PowerShell:
  PS> dbctl completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, add the output to your profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
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
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	_ = ui.Logger // ensure ui package is imported
}
