package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"micromachine.dev/cmd-utils/lib"
)

var rootDir string
var buildScript string

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		packageManager, err := lib.DetectPackageManager(&rootDir)

		if err != nil {
			lib.LogWithColor(lib.Fail, fmt.Sprintf("✗ %v", err))
			os.Exit(1)
		}

		wrangler, err := lib.DetectWranglerFile(&rootDir)

		if err != nil {
			lib.LogWithColor(lib.Fail, fmt.Sprintf("✗ %v", err))
			os.Exit(1)
		}

		entrypoint, ok := wrangler["main"].(string)

		if !ok {
			lib.LogWithColor(lib.Fail, "✗ Main entrypoint not found")
			os.Exit(1)
		}

		var assetPath string

		if assets, ok := wrangler["assets"].(map[string]any); ok {
			if dir, ok := assets["directory"].(string); ok {
				assetPath = dir
			}
		}

		bundler := lib.Bundle{
			RootDir:        rootDir,
			AssetPath:      assetPath,
			ModulePath:     entrypoint,
			PackageManager: *packageManager,
			BuildScript:    buildScript,
			WrangleConfig:  wrangler,
		}

		start := time.Now()
		lib.LogWithColor(lib.Cyan, "Running `micromachine build`...")
		bundler.RunBuildCommand()

		bundler.Pack()
		elapsed := time.Since(start)

		lib.LogWithColor(lib.Success, fmt.Sprintf("✓ Completed `micromachine build` in %s", elapsed))
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	buildCmd.PersistentFlags().StringVarP(&rootDir, "rootdir", "r", ".", "--rootdir ./apps/hello-world")
	buildCmd.PersistentFlags().StringVarP(&buildScript, "build-script", "b", "build", "--s build")
}
