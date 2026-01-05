package lib

import (
	"encoding/json"
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
}

func (b *Bundle) Pack() {
	shouldBundle := IsOpenNext(b.WrangleConfig)
	if shouldBundle {

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

		absDir, err := filepath.Abs(b.RootDir)
		if err != nil {
			LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
			os.Exit(1)
		}

		nodejsHybridPlugin := plugins.NodeJsHybridPlugin{
			BasePath:       absDir,
			PackageManager: b.PackageManager,
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

		result := api.Build(api.BuildOptions{
			Plugins: []api.Plugin{
				nodejsHybridPlugin.New(
					compatibilityDate.Format(time.DateOnly),
					[]string{"nodejs_compat", "global_fetch_strictly_public"}),
				cloudflarePlugin,
			},
			EntryPoints:   []string{strings.TrimPrefix(b.ModulePath, "/")},
			Outdir:        b.GetModuleDir(),
			AbsWorkingDir: absDir,
			Bundle:        true,
			Write:         true,
			Splitting:     false,
			LogLevel:      api.LogLevelSilent,
			Format:        api.FormatESModule,
			Platform:      api.PlatformBrowser,
			TreeShaking:   api.TreeShakingTrue,
			Loader:        map[string]api.Loader{".js": api.LoaderJSX, ".mjs": api.LoaderJSX, ".cjs": api.LoaderJSX},

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

	}

	now := time.Now()
	LogWithColor(Default, "Copying assets...")
	err := copyDir(b.RootDir+"/"+strings.TrimPrefix(b.AssetPath, "/"), b.GetAssetDir())
	if err != nil {
		LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
		os.Exit(1)
	}

	if !shouldBundle {
		err := copyDir(b.RootDir+"/"+strings.TrimPrefix(b.ModulePath, "/"), b.GetModuleDir())
		if err != nil {
			LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
			os.Exit(1)
		}
	}
	elapsed := time.Since(now)
	LogWithColor(Success, fmt.Sprintf("✓ Assets copied in %s", elapsed))
}

func (b *Bundle) RunBuildCommand() {
	// Capture stdout and stderr separately
	cmd := exec.Command(b.PackageManager, "run", b.BuildScript)
	cmd.Dir = b.RootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
		os.Exit(1)
	}
}

func (b *Bundle) GetOutputDir() string {
	return "./.micromachine"
}

func (b *Bundle) GetModuleDir() string {
	return b.GetOutputDir() + "/worker"
}

func (b *Bundle) GetAssetDir() string {
	return b.RootDir + "/.micromachine/assets"
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

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
