package cmd

import (
	"fmt"
	"runtime"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/adnanex/dbctl/pkg/ui"
)

// Build-time variables — set via ldflags during `go build`
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of dbctl",
	Long:  "Display the current version, build commit, build date, and Go runtime information.",
	Run: func(cmd *cobra.Command, args []string) {
		short, _ := cmd.Flags().GetBool("short")
		if short {
			fmt.Println(Version)
			return
		}

		title := ui.HeaderStyle.Render("dbctl")
		version := lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true).Render(Version)

		fmt.Printf("%s %s\n", title, version)
		fmt.Println()
		fmt.Printf("  %s  %s\n", ui.Faint.Render("Version:"), Version)
		fmt.Printf("  %s   %s\n", ui.Faint.Render("Commit:"), Commit)
		fmt.Printf("  %s    %s\n", ui.Faint.Render("Built:"), BuildDate)
		fmt.Printf("  %s       %s\n", ui.Faint.Render("Go:"), runtime.Version())
		fmt.Printf("  %s %s/%s\n", ui.Faint.Render("Platform:"), runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	versionCmd.Flags().Bool("short", false, "print only the version number")
	rootCmd.AddCommand(versionCmd)
}
