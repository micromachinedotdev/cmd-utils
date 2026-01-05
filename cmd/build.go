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
var buildEnv string

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Bundles the code for deployment",
	Long: `The build command automates the preparation of your application for deployment. 
It performs the following steps:
1. Detects the project's package manager (Bun, PNPM, or Yarn).
2. Locates and parses the wrangler configuration file (toml, json, or jsonc).
3. Executes the specified build script.
4. Bundles the resulting assets and entrypoints into a deployable package.`,
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
			Environment:    buildEnv,
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
	buildCmd.PersistentFlags().StringVarP(&buildScript, "build-script", "b", "build", "--b build")
	buildCmd.PersistentFlags().StringVarP(&buildEnv, "env", "e", "production", "--e production")
}
