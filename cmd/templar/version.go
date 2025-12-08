package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information - set via ldflags at build time
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, git commit, and build date of templar.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("templar %s\n", Version)
		fmt.Printf("  commit:  %s\n", GitCommit)
		fmt.Printf("  built:   %s\n", BuildDate)
		fmt.Printf("  go:      %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
