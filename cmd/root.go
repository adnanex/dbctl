package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/adnanex/dbctl/pkg/ui"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "dbctl",
	Short: "Declarative database provisioning & container orchestration",
	Long: `dbctl is a command-line tool that automates creating databases, configuring
users, managing passwords, and granting permissions declaratively from YAML files.

Define your desired database state in YAML and let dbctl converge to it
idempotently — no more brittle shell scripts or manual SQL one-liners.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Error(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Persistent flags — available to all subcommands
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "path to YAML configuration file (default: config.yaml)")
	rootCmd.PersistentFlags().String("global-config", "", "path to global configuration file (default: auto-discovers ~/.dbctl/config.yaml)")
	rootCmd.PersistentFlags().StringSliceP("env", "e", nil, "path to .env file (can be repeated)")
	rootCmd.PersistentFlags().DurationP("timeout", "t", 60_000_000_000, "overall timeout for provisioning operations") // 60s in nanoseconds
	rootCmd.PersistentFlags().Bool("dry-run", false, "simulate execution without modifying the database")
	rootCmd.PersistentFlags().Bool("skip-container", false, "skip automatic container startup/checking")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose debug output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress all output except errors")

	// Bind flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("global-config", rootCmd.PersistentFlags().Lookup("global-config"))
	viper.BindPFlag("env", rootCmd.PersistentFlags().Lookup("env"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("skip-container", rootCmd.PersistentFlags().Lookup("skip-container"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Set up logging level based on flags
	if viper.GetBool("verbose") {
		ui.SetVerbose()
	} else if viper.GetBool("quiet") {
		ui.SetQuiet()
	}

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config file in current directory and home
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")

		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(fmt.Sprintf("%s/.dbctl", home))
			viper.AddConfigPath(fmt.Sprintf("%s/.config/dbctl", home))
		}
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("DBCTL")
	viper.AutomaticEnv()

	// If a config file is found, read it in (don't error if not found — provision cmd handles that)
	viper.ReadInConfig()
}
