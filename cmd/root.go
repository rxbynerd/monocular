package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagURL       string
	flagDirectory string
	flagFilter    string
	flagJSON      bool
	flagNoColor   bool
	appVersion    string
)

func SetVersion(v string) {
	appVersion = v
}

var rootCmd = &cobra.Command{
	Use:   "monocular",
	Short: "Real-time TUI dashboard for OpenCode SSE events",
	Long: `Monocular connects to a running OpenCode server's SSE event stream
and presents a real-time visual dashboard of everything happening
in the instance. This is a read-only diagnostic/observability tool.`,
	Version: "dev",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := Config{
			URL:       flagURL,
			Directory: flagDirectory,
			Filter:    parseFilter(flagFilter),
			JSON:      flagJSON,
			NoColor:   flagNoColor,
		}

		if cfg.JSON {
			fmt.Println("JSON mode not yet implemented")
			return nil
		}

		fmt.Println("TUI mode not yet implemented")
		return nil
	},
}

type Config struct {
	URL       string
	Directory string
	Filter    []string
	JSON      bool
	NoColor   bool
}

func parseFilter(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func init() {
	rootCmd.Flags().StringVarP(&flagURL, "url", "u", "http://127.0.0.1:4096", "OpenCode server URL")
	rootCmd.Flags().StringVarP(&flagDirectory, "directory", "d", "", "Initial client-side directory filter")
	rootCmd.Flags().StringVarP(&flagFilter, "filter", "f", "", "Comma-separated event categories to show (default: all)\nCategories: session,message,permission,question,file,infra,pty,workspace,tui,todo")
	rootCmd.Flags().BoolVar(&flagJSON, "json", false, "Output raw events as NDJSON to stdout (no TUI)")
	rootCmd.Flags().BoolVar(&flagNoColor, "no-color", false, "Disable colors")
}

func Execute() error {
	rootCmd.Version = appVersion
	return rootCmd.Execute()
}
