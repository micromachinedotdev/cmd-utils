package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"micromachine.dev/cmd-utils/lib/bundler"
	"micromachine.dev/cmd-utils/lib/utils"
)

var rootDir string
var buildScript string
var buildEnv string
var userDefinedEntrypoint string
var shouldBundle bool

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
		packageManager, err := utils.DetectPackageManager(&rootDir)

		if err != nil {
			utils.LogWithColor(utils.Fail, fmt.Sprintf("✗ %v", err))
			os.Exit(1)
		}

		wrangler, err := utils.DetectWranglerFile(&rootDir)

		if err != nil {
			utils.LogWithColor(utils.Fail, fmt.Sprintf("✗ %v", err))
			os.Exit(2)
		}

		wranglerEntrypoint, _ := wrangler["main"].(string)

		var assetPath string

		if assets, ok := wrangler["assets"].(map[string]any); ok {
			if dir, ok := assets["directory"].(string); ok {
				assetPath = dir
			}
		}

		var entrypoint string

		if wranglerEntrypoint != "" {
			entrypoint = wranglerEntrypoint
		}

		if userDefinedEntrypoint != "" {
			entrypoint = userDefinedEntrypoint
		}

		if entrypoint == "" {
			utils.LogWithColor(utils.Fail, "✗ No entrypoint not found")
			os.Exit(2)
		}

		bundle := bundler.Bundle{
			RootDir:        rootDir,
			AssetPath:      assetPath,
			ModulePath:     entrypoint,
			PackageManager: *packageManager,
			BuildScript:    buildScript,
			Environment:    buildEnv,
			WrangleConfig:  wrangler,
			ShouldBundle:   shouldBundle,
		}

		start := time.Now()
		utils.LogWithColor(utils.Cyan, "Running `micromachine build`...")

		switch true {
		case utils.IsNextJS(rootDir):
			// Run open-next-build
			start := time.Now()
			utils.LogWithColor(utils.Default, "Running `opennextjs-cloudflare build`...")

			// Install opennextjs/cloudflare for next
			err := bundle.RunCommand(*packageManager, "--silent", "install", "@opennextjs/cloudflare")
			if err != nil {
				utils.LogWithColor(utils.Fail, fmt.Sprintf("✗ %v", err))
				os.Exit(1)
			}

			// Run build
			err = bundle.RunCommand(*packageManager, "opennextjs-cloudflare", "build")
			if err != nil {
				utils.LogWithColor(utils.Fail, fmt.Sprintf("✗ %v", err))
				os.Exit(1)
			}

			// Calculate time elapsed.
			elapsed := time.Since(start)
			utils.LogWithColor(utils.Success, fmt.Sprintf("✓ Completed `opennextjs-cloudflare build` in %s", elapsed))
		default:
			if bundle.BuildScript != "" {
				bundle.RunBuildCommand()
			}
		}

		bundle.Pack()
		elapsed := time.Since(start)

		utils.LogWithColor(utils.Success, fmt.Sprintf("✓ Completed `micromachine build` in %s", elapsed))
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
	buildCmd.PersistentFlags().StringVarP(&rootDir, "rootdir", "r", ".", "--rootdir ./apps/client")
	buildCmd.PersistentFlags().StringVarP(&buildScript, "script", "s", "", "--s build")
	buildCmd.PersistentFlags().StringVarP(&buildEnv, "env", "e", "production", "--e production")
	buildCmd.PersistentFlags().StringVarP(&userDefinedEntrypoint, "entrypoint", "i", "", "--i ./dist/worker.js")
	buildCmd.PersistentFlags().BoolVarP(&shouldBundle, "bundle", "b", false, "--bundle")
}
