package bundler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"micromachine.dev/cmd-utils/lib/bundler/plugins"
	"micromachine.dev/cmd-utils/lib/utils"
)

type Bundle struct {
	RootDir             string
	ModulePath          string
	PackageManager      string
	AssetPath           string
	BuildScript         string
	Environment         string
	WranglerConfig      *utils.WranglerConfig
	BuildWranglerConfig *utils.NormalizedWranglerConfig
	ShouldBundle        bool
}

func (b *Bundle) Pack() error {
	absDir, err := filepath.Abs(b.RootDir)
	if err != nil {
		slog.Error(fmt.Sprintf("✗ %v", err))
		return fmt.Errorf("could not resolve absolute path: %w", err)
	}

	err = os.MkdirAll(filepath.Join(absDir, b.GetOutputDir()), 0755)

	if err != nil {
		slog.Error(fmt.Sprintf("✗ %v", err))
		return fmt.Errorf("could not create output directory: %w", err)
	}

	shouldBundle := utils.IsOpenNext(b.WranglerConfig) || b.ShouldBundle

	if b.BuildWranglerConfig != nil {
		shouldBundle = !b.BuildWranglerConfig.NoBundle
	}

	modulePath := b.ModulePath
	if b.BuildWranglerConfig != nil && b.BuildWranglerConfig.Main != "" {
		outBase := b.findModuleDir()
		modulePath = filepath.Join(outBase, b.BuildWranglerConfig.Main)
	}

	if shouldBundle {
		start := time.Now()
		utils.LogWithColor(utils.Cyan, "Bundling application...")

		var warnedPackages = make(map[string][]string)

		var cfPaths = make(map[string]struct{})

		cloudflarePlugin := api.Plugin{
			Name: "cloudflare-internal",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: "^cloudflare:"},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						cfPaths[args.Path] = struct{}{}
						return api.OnResolveResult{External: true}, nil
					})
			},
		}

		nodejsHybridPlugin := plugins.NodeJsHybridPlugin{
			BasePath:       absDir,
			PackageManager: b.PackageManager,
		}

		externalFilesPlugin := plugins.ExternalFilePlugin{
			Extensions: []string{
				".wasm",
				".bin",
				".html",
				".txt",
			},
		}

		wranglerCDate := b.WranglerConfig.CompatibilityDate
		if b.BuildWranglerConfig != nil && b.BuildWranglerConfig.CompatibilityDate != "" {
			wranglerCDate = b.BuildWranglerConfig.CompatibilityDate
		}

		compatibilityDate, err := time.Parse(time.DateOnly, wranglerCDate)
		if err != nil {
			slog.Error("✗ Invalid compatibility_date: must be in format YYYY-MM-DD")
			return fmt.Errorf("invalid compatibility_date format: %w", err)
		}

		compatibilityFlags := b.WranglerConfig.CompatibilityFlags
		if b.BuildWranglerConfig != nil && b.BuildWranglerConfig.CompatibilityFlags != nil {
			compatibilityFlags = b.BuildWranglerConfig.CompatibilityFlags
		}

		external := []string{"__STATIC_CONTENT_MANIFEST"}

		result := api.Build(api.BuildOptions{
			Plugins: []api.Plugin{
				nodejsHybridPlugin.New(
					compatibilityDate.Format(time.DateOnly),
					compatibilityFlags,
				),
				externalFilesPlugin.New(),
				cloudflarePlugin,
			},
			EntryPoints:    []string{modulePath},
			Outdir:         b.GetModuleDir(),
			AbsWorkingDir:  absDir,
			Bundle:         true,
			Write:          true,
			AllowOverwrite: true,
			Splitting:      true,
			// LogLevel:       api.LogLevelInfo,
			Format:      api.FormatESModule,
			Platform:    api.PlatformNeutral,
			TreeShaking: api.TreeShakingTrue,
			Loader:      map[string]api.Loader{".js": api.LoaderJSX, ".mjs": api.LoaderJSX, ".cjs": api.LoaderJSX},

			// Target modern runtime (Cloudflare Workers)
			Target:            api.ESNext,
			External:          external,
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
			KeepNames:         true,
			Metafile:          false,
			Sourcemap:         api.SourceMapLinked,
			Conditions:        []string{"workerd", "worker", "browser"},
			Define: map[string]string{
				"process.env.NODE_ENV":            toJSString(b.Environment),
				"global.process.env.NODE_ENV":     toJSString(b.Environment),
				"globalThis.process.env.NODE_ENV": toJSString(b.Environment),
			},
		})

		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				slog.Error(fmt.Sprintf("✗ %v", err))
			}

			return fmt.Errorf("bundle failed with %d error(s)", len(result.Errors))
		}

		// Later, after build:
		for path, importers := range warnedPackages {
			slog.Warn(fmt.Sprintf("WARN! Node builtin %q used (from %v)\n", path, importers))
		}

		elapsed := time.Since(start)
		utils.LogWithColor(utils.Success, fmt.Sprintf("✓ Bundling completed in %s", elapsed))
	} else {
		err = os.MkdirAll(filepath.Join(absDir, b.GetModuleDir()), 0755)
		if err != nil {
			slog.Error(fmt.Sprintf("✗ %v", err))
			return fmt.Errorf("could not create module directory: %w", err)
		}

		err = copyDir(filepath.Dir(filepath.Join(absDir, modulePath)), filepath.Join(absDir, b.GetModuleDir()), []string{})
		if err != nil {
			slog.Error("✗ Could not copy module files", slog.Any("error", err))
			return fmt.Errorf("could not copy module files: %w", err)
		}
	}

	now := time.Now()

	if b.BuildWranglerConfig != nil && b.BuildWranglerConfig.Assets != nil && b.BuildWranglerConfig.Assets.Directory != "" {
		dir := filepath.Join(absDir, filepath.Dir(modulePath), b.BuildWranglerConfig.Assets.Directory)
		if _, err := os.Stat(dir); err == nil {

			utils.LogWithColor(utils.Default, "Copying assets...")

			modulePathDir := filepath.Join(absDir, filepath.Dir(modulePath))
			err = copyDir(dir, b.GetAssetDir(), []string{modulePathDir})
			if err != nil {
				slog.Error(fmt.Sprintf("✗ %v", err))
				return fmt.Errorf("could not copy assets: %w", err)
			}

			elapsed := time.Since(now)
			utils.LogWithColor(utils.Success, fmt.Sprintf("✓ Assets copied in %s", elapsed))
		} else if !errors.Is(err, os.ErrNotExist) {
			slog.Error(fmt.Sprintf("✗ %v", err))
			return fmt.Errorf("could not stat assets directory: %w", err)
		}
	} else if utils.HasAssets(b.WranglerConfig) && b.AssetPath != "" {
		if _, err := os.Stat(filepath.Join(absDir, b.AssetPath)); err == nil {

			utils.LogWithColor(utils.Default, "Copying assets...")

			modulePathDir := filepath.Join(absDir, filepath.Dir(b.ModulePath))
			err = copyDir(filepath.Join(absDir, strings.TrimPrefix(b.AssetPath, "/")), b.GetAssetDir(), []string{modulePathDir})

			if err != nil {
				slog.Error(fmt.Sprintf("✗ %v", err))
				return fmt.Errorf("could not copy assets: %w", err)
			}
			elapsed := time.Since(now)
			utils.LogWithColor(utils.Success, fmt.Sprintf("✓ Assets copied in %s", elapsed))

		} else if !errors.Is(err, os.ErrNotExist) {
			slog.Error(fmt.Sprintf("✗ %v", err))
			return fmt.Errorf("could not stat assets directory: %w", err)
		}
	}

	return nil
}

func (b *Bundle) RunBuildCommand() error {
	absDir, err := filepath.Abs(b.RootDir)
	if err != nil {
		slog.Error(fmt.Sprintf("✗ %v", err))
		return fmt.Errorf("could not resolve absolute path: %w", err)
	}
	costomConfig := utils.IncludeCloudflareVitePlugin(absDir, b.PackageManager)

	cmdName := b.PackageManager + " run " + b.BuildScript

	if costomConfig != nil {
		cmdName = fmt.Sprintf("%s --config %s", cmdName, *costomConfig)
	}

	start := time.Now()
	utils.LogWithColor(utils.Default, fmt.Sprintf("Running `%s`...", cmdName))
	err = b.RunCommand(b.PackageManager, "run", b.BuildScript)

	if err != nil {
		slog.Error(fmt.Sprintf("✗ %v", err))
		return fmt.Errorf("build command failed: %w", err)
	}
	elapsed := time.Since(start)
	utils.LogWithColor(utils.Success, fmt.Sprintf("✓ Completed `%s` in %s", cmdName, elapsed))

	outputDir := filepath.Join(absDir, b.findModuleDir())
	if outputDir != "" {
		wrangler, err := utils.DetectWranglerFile[utils.NormalizedWranglerConfig](&outputDir)
		if err != nil {
			slog.Error(fmt.Sprintf("✗ %v", err))
			return err
		}

		b.BuildWranglerConfig = wrangler
	}

	return nil
}

func (b *Bundle) findModuleDir() string {
	absDir, err := filepath.Abs(b.RootDir)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	mainDir := filepath.Dir(b.WranglerConfig.Main)

	paths := []string{
		mainDir,
		"dist/server",
		".output/server",
	}

	// Check for generated deployment config (from framework builds)
	deployConfigPath := filepath.Join(absDir, ".wrangler/deploy/config.json")
	if data, err := os.ReadFile(deployConfigPath); err == nil {
		var deployConfig struct {
			ConfigPath string `json:"configPath"`
		}
		if json.Unmarshal(data, &deployConfig) == nil && deployConfig.ConfigPath != "" {
			// Prepend the directory containing the generated config
			generatedDir := filepath.Dir(filepath.Join(absDir, deployConfig.ConfigPath))
			paths = slices.Insert(paths, 0, generatedDir)
		}
	}

	for _, p := range paths {
		pathWithAbsDir := filepath.Join(absDir, p)
		if info, err := os.Stat(pathWithAbsDir); err == nil && info.IsDir() {
			return p
		}
	}

	return mainDir
}

func (b *Bundle) RunExecutableCommand(args ...string) error {
	switch b.PackageManager {
	case "npm":
		return b.RunCommand("npx", args...)
	case "yarn":
		args = append([]string{"dlx"}, args...)
		return b.RunCommand("yarn", args...)
	case "pnpm":
		args = append([]string{"dlx"}, args...)
		return b.RunCommand("pnpm", args...)
	case "bun":
		return b.RunCommand("bunx", args...)
	}

	return fmt.Errorf("invalid package manager %s", b.PackageManager)
}

func (b *Bundle) RunCommand(name string, args ...string) error {
	// Capture stdout and stderr separately
	cmd := exec.Command(name, args...)
	cmd.Dir = b.RootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		return err
	}

	return nil
}

func (b *Bundle) GetOutputDir() string {
	return "./.micromachine"
}

func (b *Bundle) GetModuleDir() string {
	return filepath.Join(b.GetOutputDir(), "/worker")
}

func (b *Bundle) GetAssetDir() string {
	return filepath.Join(b.RootDir, ".micromachine/assets")
}

func copyDir(src, dst string, ignorePath []string) error {

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		for _, p := range ignorePath {
			pathRel, err := filepath.Rel(src, p)
			if err != nil {
				continue
			}

			if strings.HasPrefix(rel, pathRel) {
				return nil
			}
		}

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {

			return err
		}

		info, _ := d.Info()
		return os.WriteFile(target, data, info.Mode())
	})
}

func toJSString(val string) string {
	if val == "" {
		return `""` // or "undefined" depending on your needs
	}
	// JSON marshal handles escaping
	b, _ := json.Marshal(val)
	return string(b)
}
