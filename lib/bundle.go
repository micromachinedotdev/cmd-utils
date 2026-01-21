package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"micromachine.dev/cmd-utils/lib/plugins"
)

type Bundle struct {
	RootDir        string
	ModulePath     string
	PackageManager string
	AssetPath      string
	BuildScript    string
	Environment    string
	WrangleConfig  map[string]any
	ShouldBundle   bool
}

func (b *Bundle) Pack() {
	absDir, err := filepath.Abs(b.RootDir)
	if err != nil {
		LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
		os.Exit(1)
	}

	err = os.MkdirAll(filepath.Join(absDir, b.GetOutputDir()), 0755)

	if err != nil {
		LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
		os.Exit(1)
	}

	shouldBundle := IsOpenNext(b.WrangleConfig) || b.ShouldBundle

	if shouldBundle {
		start := time.Now()
		LogWithColor(Cyan, "Bundling application...")

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

		wranglerCDate, ok := b.WrangleConfig["compatibility_date"].(string)

		if !ok {
			LogWithColor(Fail, "✗ Invalid wrangler configuration: missing or invalid compatibility_date")
			os.Exit(1)
		}

		compatibilityDate, err := time.Parse(time.DateOnly, wranglerCDate)
		if err != nil {
			LogWithColor(Fail, "✗ Invalid compatibility_date: must be in format YYYY-MM-DD")
			os.Exit(1)
		}

		compatibilityFlags := make([]string, 0)

		if b.WrangleConfig["compatibility_flags"] != nil {
			if flags, ok := b.WrangleConfig["compatibility_flags"].([]any); ok {
				for _, v := range flags {
					if flag, ok := v.(string); ok {
						compatibilityFlags = append(compatibilityFlags, flag)
					}
				}

			}
		}

		fmt.Println(b.WrangleConfig)
		result := api.Build(api.BuildOptions{
			Plugins: []api.Plugin{
				nodejsHybridPlugin.New(compatibilityDate.Format(time.DateOnly), compatibilityFlags),
				externalFilesPlugin.New(),
				cloudflarePlugin,
			},
			EntryPoints:   []string{strings.TrimPrefix(b.ModulePath, "/")},
			Outdir:        b.GetModuleDir(),
			AbsWorkingDir: absDir,
			Bundle:        true,
			Write:         true,
			// AllowOverwrite: true,
			Splitting: false,
			// LogLevel:       api.LogLevelSilent,
			Format:      api.FormatESModule,
			Platform:    api.PlatformNeutral,
			TreeShaking: api.TreeShakingTrue,
			Loader:      map[string]api.Loader{".js": api.LoaderJSX, ".mjs": api.LoaderJSX, ".cjs": api.LoaderJSX},

			// Target modern runtime (Cloudflare Workers)
			Target: api.ESNext,

			External:          []string{"__STATIC_CONTENT_MANIFEST"},
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
			KeepNames:         true,
			Metafile:          true,
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
				LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
			}

			os.Exit(1)
		}

		// Later, after build:
		for path, importers := range warnedPackages {
			LogWithColor(Warning, fmt.Sprintf("WARN! Node builtin %q used (from %v)\n", path, importers))
		}
		elapsed := time.Since(start)
		LogWithColor(Success, fmt.Sprintf("✓ Bundling completed in %s", elapsed))
	}

	now := time.Now()

	if HasAssets(b.WrangleConfig) && b.AssetPath != "" {
		if _, err := os.Stat(filepath.Join(absDir, b.AssetPath)); err == nil {
			LogWithColor(Default, "Copying assets...")
			modulePathDir := filepath.Join(absDir, filepath.Dir(b.ModulePath))
			err = copyDir(filepath.Join(absDir, strings.TrimPrefix(b.AssetPath, "/")), b.GetAssetDir(), []string{modulePathDir})

			if err != nil {
				LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
				os.Exit(1)
			}
			elapsed := time.Since(now)
			LogWithColor(Success, fmt.Sprintf("✓ Assets copied in %s", elapsed))
		} else if !errors.Is(err, os.ErrNotExist) {
			LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
			os.Exit(1)
		}
	}
}

func (b *Bundle) RunBuildCommand() {
	cmdName := b.PackageManager + " run " + b.BuildScript
	start := time.Now()
	LogWithColor(Default, fmt.Sprintf("Running `%s`...", cmdName))
	err := b.RunCommand(b.PackageManager, "run", b.BuildScript)

	if err != nil {
		LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
		os.Exit(1)
	}
	elapsed := time.Since(start)
	LogWithColor(Success, fmt.Sprintf("✓ Completed `%s` in %s", cmdName, elapsed))
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
