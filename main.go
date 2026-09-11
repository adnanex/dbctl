package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/adnanex/dbctl/pkg/config"
	"github.com/adnanex/dbctl/pkg/container"
	"github.com/adnanex/dbctl/pkg/driver"

	_ "github.com/adnanex/dbctl/pkg/driver/mongodb"
	_ "github.com/adnanex/dbctl/pkg/driver/mysql"
	_ "github.com/adnanex/dbctl/pkg/driver/postgres"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	parts := strings.Split(value, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			*i = append(*i, p)
		}
	}
	return nil
}

func handleInitCommand(args []string) {
	initFlags := flag.NewFlagSet("init", flag.ExitOnError)
	templateType := initFlags.String("type", "", "Template type: 'global', 'local', or 'minimal' (defaults based on destination)")
	initFlags.StringVar(templateType, "t", "", "Template type (shorthand)")
	force := initFlags.Bool("force", false, "Overwrite existing configuration file if it exists")
	initFlags.BoolVar(force, "f", false, "Overwrite existing file (shorthand)")

	initFlags.Usage = func() {
		fmt.Println("Usage: dbctl init [destination] [options]")
		fmt.Println("\nDestinations:")
		fmt.Println("  local / .       Create ./config.yaml (default)")
		fmt.Println("  global / ~      Create ~/.dbctl/config.yaml")
		fmt.Println("  config / .config Create ~/.config/dbctl/config.yaml")
		fmt.Println("  <filepath>      Create template at custom filepath")
		fmt.Println("\nOptions:")
		initFlags.PrintDefaults()
	}

	var flagArgs []string
	var posArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if (arg == "-type" || arg == "-t" || arg == "--type" || arg == "-path" || arg == "-p") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}

	if err := initFlags.Parse(flagArgs); err != nil {
		os.Exit(1)
	}

	dest := "local"
	if len(posArgs) > 0 {
		dest = posArgs[0]
	} else if initFlags.NArg() > 0 {
		dest = initFlags.Arg(0)
	}

	targetPath, isGlobal, err := config.ResolveInitPath(dest)
	if err != nil {
		log.Fatalf("Error resolving path: %v", err)
	}

	tpl := *templateType
	if tpl == "" {
		if isGlobal {
			tpl = "global"
		} else {
			tpl = "local"
		}
	}

	if err := config.GenerateConfigFile(targetPath, tpl, *force); err != nil {
		log.Fatalf("Error initializing config: %v", err)
	}

	log.Printf("Successfully created %s template configuration at: %s", tpl, targetPath)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		handleInitCommand(os.Args[2:])
		return
	}

	var (
		configFile    string
		globalConfig  string
		dryRun        bool
		skipContainer bool
		timeout       time.Duration
		envFiles      arrayFlags
	)

	flag.StringVar(&configFile, "config", "config.yaml", "Path to YAML configuration file")
	flag.StringVar(&configFile, "c", "config.yaml", "Path to YAML configuration file (shorthand)")
	flag.StringVar(&globalConfig, "global-config", "", "Path to global configuration file (default auto-discovers ~/.dbctl/config.yaml)")
	flag.StringVar(&globalConfig, "g", "", "Path to global configuration file (shorthand)")
	flag.Var(&envFiles, "env", "Path to env file (can be repeated, e.g. -env .env.mysql -env ./.env)")
	flag.Var(&envFiles, "e", "Path to env file (shorthand, can be repeated)")
	flag.BoolVar(&dryRun, "dry-run", false, "Simulate execution without modifying the database")
	flag.BoolVar(&skipContainer, "skip-container", false, "Skip automatic container startup/checking")
	flag.DurationVar(&timeout, "timeout", 60*time.Second, "Overall timeout for provisioning operations")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[dbctl] ")

	// If no explicit env files provided, check if .env exists in current working directory
	if len(envFiles) == 0 {
		if _, err := os.Stat(".env"); err == nil {
			log.Printf("Detected local .env file. Automatically loading...")
			envFiles = append(envFiles, ".env")
		}
	}

	var envMap map[string]string
	if len(envFiles) > 0 {
		log.Printf("Loading environment variables from: %v", []string(envFiles))
		var err error
		envMap, err = config.LoadEnvFiles(envFiles)
		if err != nil {
			log.Fatalf("Error loading env file(s): %v", err)
		}
		log.Printf("Loaded %d variable(s) from env file(s)", len(envMap))
	}

	// 1. Discover and load global configuration (if present)
	globalPath := config.FindGlobalConfigFile(globalConfig)
	var globalCfg *config.GlobalConfig
	if globalPath != "" {
		log.Printf("Loading global configuration from %s...", globalPath)
		var err error
		globalCfg, err = config.LoadGlobalConfig(globalPath, envMap)
		if err != nil {
			log.Printf("Warning: failed to load global config from %s: %v", globalPath, err)
		} else {
			log.Printf("Loaded %d driver default(s) from global config", len(globalCfg.Defaults))
		}
	}

	// 2. Load project configuration
	log.Printf("Loading configuration from %s...", configFile)
	cfg, err := config.LoadConfig(configFile, envMap)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// 3. Merge global defaults into target configurations
	if globalCfg != nil {
		config.MergeGlobalDefaults(cfg, globalCfg)
	}

	if len(cfg.Targets) == 0 {
		log.Fatalf("No target databases defined in %s", configFile)
	}

	log.Printf("Found %d target database(s) to process. Supported drivers: %v",
		len(cfg.Targets), driver.SupportedDrivers())

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	successCount := 0
	for i, target := range cfg.Targets {
		fmt.Printf("\n--- [%d/%d] Processing %s on %s:%d ---\n",
			i+1, len(cfg.Targets), target.Driver, target.Host, target.Port)

		// Check and start container if configured
		if !skipContainer && !dryRun && target.Container != nil {
			if err := container.EnsureContainerRunning(ctx, target.Container, target.Driver, target.Host, target.Port); err != nil {
				log.Printf("ERROR starting container for %s: %v", target.Driver, err)
				os.Exit(1)
			}
		}

		if err := driver.ProvisionTarget(ctx, &target, dryRun); err != nil {
			log.Printf("ERROR on target %s (%s:%d): %v", target.Driver, target.Host, target.Port, err)
			os.Exit(1)
		}
		successCount++
	}

	fmt.Println()
	log.Printf("Done! Successfully provisioned %d/%d database target(s).", successCount, len(cfg.Targets))
}
