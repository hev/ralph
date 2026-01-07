package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/hev/ralph/internal/config"
	"github.com/hev/ralph/internal/runner"
	"github.com/spf13/cobra"
)

var cfg = config.DefaultConfig()

var rootCmd = &cobra.Command{
	Use:   "ralph",
	Short: "Run Claude in a loop",
	Long: fmt.Sprintf(`ralph v%s - Run Claude in a loop

"I'm in danger!" - Ralph Wiggum`, config.Version),
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Printf("ralph v%s\n", config.Version)
			return nil
		}
		return runner.Run(cfg)
	},
}

var showVersion bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ralph v%s\n", config.Version)
	},
}

func init() {
	// Core options (matching bash version)
	rootCmd.Flags().StringVarP(&cfg.PromptFile, "prompt", "p", cfg.PromptFile, "Path to prompt file")
	rootCmd.Flags().IntVarP(&cfg.MaxIterations, "max-iterations", "n", cfg.MaxIterations, "Max loop iterations (0 = unlimited)")
	rootCmd.Flags().IntVarP(&cfg.MaxTime, "max-time", "t", cfg.MaxTime, "Max total runtime in seconds (0 = unlimited)")
	rootCmd.Flags().StringVarP(&cfg.AgentDir, "agent-dir", "d", cfg.AgentDir, "Scratchpad directory")
	rootCmd.Flags().IntVarP(&cfg.Cooldown, "cooldown", "c", cfg.Cooldown, "Delay between iterations")
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "", true, "Enable verbose output")
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "quiet", "q", false, "Disable verbose output")
	rootCmd.Flags().BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "Show what would run without executing")

	// OTEL options
	rootCmd.Flags().BoolVar(&cfg.OTELEnabled, "otel-enabled", cfg.OTELEnabled, "Enable metrics export")
	rootCmd.Flags().StringVar(&cfg.OTELEndpoint, "otel-endpoint", cfg.OTELEndpoint, "OTLP endpoint")
	rootCmd.Flags().StringVar(&cfg.MetricsPrefix, "metrics-prefix", cfg.MetricsPrefix, "Metric name prefix")
	rootCmd.Flags().StringVar(&cfg.ProjectName, "project-name", cfg.ProjectName, "Override project label")

	// Handle -q flag properly (inverts verbose)
	rootCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("quiet") {
			cfg.Verbose = false
		}
	}

	// Add -v flag for version
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version")

	rootCmd.AddCommand(versionCmd)

	// Custom help template to match bash version style
	rootCmd.SetHelpTemplate(fmt.Sprintf(`ralph v%s - Run Claude in a loop

Usage: ralph [OPTIONS]

Options:
  -h, --help                  Show this help message
  -p, --prompt FILE           Path to prompt file (default: ./prompt.md)
  -n, --max-iterations N      Max loop iterations (default: 0 = unlimited)
  -t, --max-time SECONDS      Max total runtime in seconds (default: 0 = unlimited)
  -d, --agent-dir DIR         Scratchpad directory (default: ./.agent)
  -c, --cooldown SECONDS      Delay between iterations (default: 1)
  -q, --quiet                 Disable verbose output
  --dry-run                   Show what would run without executing

OTEL Options:
  --otel-enabled              Enable metrics export (default: false)
  --otel-endpoint URL         OTLP endpoint (default: localhost:4317)
  --metrics-prefix PREFIX     Metric name prefix (default: ralph)
  --project-name NAME         Override project label (default: cwd basename)

Examples:
  ralph                           # Run forever with defaults
  ralph -n 5                      # Run for 5 iterations
  ralph -t 3600                   # Run for 1 hour
  ralph -p ~/tasks/build.md       # Use custom prompt file
  ralph -n 10 -c 5                # 10 iterations, 5s cooldown

"I'm in danger!" - Ralph Wiggum
`, config.Version))
}

func main() {
	// Disable color output if not a terminal
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		color.NoColor = true
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
