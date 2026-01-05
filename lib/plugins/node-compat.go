package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

const requiredNodeBuiltInNamespace = "node-built-in-modules"
const requiredUnenvAliasNamespace = "required-unenv-alias"

type NodeJsHybridPlugin struct {
	BasePath       string
	PackageManager string
}

var nodeModulesReStr = fmt.Sprintf(`^(node:)?(%s)$`, strings.Join(getNodeJsBuiltinModules(), "|"))
var nodeModulesRe = regexp.MustCompile(nodeModulesReStr)

func (p *NodeJsHybridPlugin) New(compatibilityDate string, compatibilityFlags []string) api.Plugin {
	return api.Plugin{
		Name: "hybrid-nodejs_compat",
		Setup: func(build api.PluginBuild) {

			cfg, err := p.getUnenvConfig(p.BasePath, compatibilityDate, compatibilityFlags)

			if err != nil {
				panic(err)
			}

			p.errorOnServiceWorkerFormat(build)
			p.handleRequireCallsToNodeJSBuiltins(build)
			p.handleUnenvAliasedPackages(build, cfg.Alias, cfg.External)
			p.handleNodeJSGlobals(build, cfg.Inject, cfg.Polyfill)
		},
	}
}

/**
 * If we are bundling a "Service Worker" formatted Worker, imports of external modules,
 * which won't be inlined/bundled by esbuild, are invalid.
 *
 * This `onResolve()` handler will error if it identifies node.js external imports.
 */
func (*NodeJsHybridPlugin) errorOnServiceWorkerFormat(build api.PluginBuild) {
	paths := make(map[string]string)

	build.OnStart(func() (api.OnStartResult, error) {
		paths = make(map[string]string)
		return api.OnStartResult{}, nil
	})

	build.OnResolve(api.OnResolveOptions{Filter: nodeModulesReStr}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		if nodeModulesRe.MatchString(regexp.QuoteMeta(args.Path)) {
			paths[args.Path] = args.Importer
		}
		return api.OnResolveResult{}, nil
	})

	build.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {

		if build.InitialOptions.Format == api.FormatIIFE && len(paths) > 0 {
			pathList := make([]string, 0, len(paths))
			for path := range paths {
				pathList = append(pathList, path)
			}
			return api.OnEndResult{
				Errors: []api.Message{
					{
						Text: fmt.Sprintf(`Unexpected external import of %[3]s.
							Your worker has no default export, which means it is assumed to be a Service Worker format Worker.
							Did you mean to create a ES Module format Worker?
							If so, try adding %[1]cexport default { ... }%[2]c in your entry-point.
							See https://developers.cloudflare.com/workers/reference/migrate-to-module-workers/.`, '`', '`', strings.Join(pathList, ", ")),
					},
				},
			}, nil
		}
		return api.OnEndResult{}, nil
	})
}

/**
 * We must convert `require()` calls for Node.js modules to a virtual ES Module that can be imported avoiding the require calls.
 * We do this by creating a special virtual ES module that re-exports the library in an onLoad handler.
 * The onLoad handler is triggered by matching the "namespace" added to the resolve.
 */

func (*NodeJsHybridPlugin) handleRequireCallsToNodeJSBuiltins(build api.PluginBuild) {
	build.OnResolve(api.OnResolveOptions{Filter: nodeModulesReStr}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		if args.Kind == api.ResolveJSRequireCall {
			return api.OnResolveResult{
				Namespace: requiredNodeBuiltInNamespace,
				Path:      args.Path,
			}, nil
		}

		return api.OnResolveResult{}, nil
	})

	build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: requiredNodeBuiltInNamespace}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		contents := fmt.Sprintf(`import libDefault from '%s';
					module.exports = libDefault;`, args.Path)
		return api.OnLoadResult{
			Contents: &contents,
			Loader:   api.LoaderJS,
		}, nil
	})
}

/**
 * Handles aliased NPM packages.
 *
 * @param build ESBuild PluginBuild.
 * @param alias Aliases resolved to absolute paths.
 * @param external external modules.
 */

func (p *NodeJsHybridPlugin) handleUnenvAliasedPackages(build api.PluginBuild, alias map[string]string, external []string) {
	aliasAbsolutePaths := make(map[string]string)
	for alias, path := range alias {
		resolved, err := nodeResolve(p.BasePath, path)

		if err != nil {
			continue
		}

		aliasAbsolutePaths[alias] = *resolved
	}

	keys := make([]string, 0, len(aliasAbsolutePaths))
	for k := range aliasAbsolutePaths {
		keys = append(keys, regexp.QuoteMeta(k))
	}

	unenvAliasReStr := fmt.Sprintf(`^(%s)$`, strings.Join(keys, "|"))

	build.OnResolve(api.OnResolveOptions{Filter: unenvAliasReStr}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		unresolvedAlias := alias[args.Path]
		if args.Kind == api.ResolveJSRequireCall &&
			(strings.HasPrefix(unresolvedAlias, "unenv/npm/") ||
				strings.HasPrefix(unresolvedAlias, "unenv/mock/")) {
			return api.OnResolveResult{
				Namespace: requiredUnenvAliasNamespace,
				Path:      unresolvedAlias,
			}, nil
		}

		return api.OnResolveResult{
			Path:     aliasAbsolutePaths[args.Path],
			External: slices.Contains(external, unresolvedAlias),
		}, nil
	})

	build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: requiredUnenvAliasNamespace}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		content := `
					import * as esm from '${path}';
					module.exports = Object.entries(esm)
								.filter(([k,]) => k !== 'default')
								.reduce((cjs, [k, value]) =>
									Object.defineProperty(cjs, k, { value, enumerable: true }),
									"default" in esm ? esm.default : {}
								);
				`
		return api.OnLoadResult{
			Contents: &content,
			Loader:   api.LoaderJS,
		}, nil
	})
}

/**
 * Inject node globals defined in unenv's preset `inject` and `polyfill` properties.
 *
 * - an `inject` injects virtual module defining the name on `globalThis`
 * - a `polyfill` is injected directly
 */

type moduleIdentifier struct {
	injectedName string
	exportName   string
	importName   string
}

func (p *NodeJsHybridPlugin) handleNodeJSGlobals(build api.PluginBuild, inject map[string]any, polyfill []string) {
	unenvVirtualModuleReStr := "_virtual_unenv_global_polyfill-(.+)$"

	prefix := pathResolve(p.BasePath, "_virtual_unenv_global_polyfill-")

	/**
	 * Map of module identifiers to
	 * - `injectedName`: the name injected on `globalThis`
	 * - `exportName`: the export name from the module
	 * - `importName`: the imported name
	 */

	injectsByModule := make(map[string][]moduleIdentifier)

	// Module specifier (i.e. `/unenv/runtime/node/...`) keyed by path (i.e. `/prefix/_virtual_unenv_global_polyfill-...`)
	virtualModulePathToSpecifier := make(map[string]string)

	for injectName, moduleSpecifier := range inject {
		var module, exportName, importName string

		if arr, ok := moduleSpecifier.([]interface{}); ok {
			strings := make([]string, len(arr))
			for i, v := range arr {
				strings[i] = v.(string)
			}
			module = strings[0]
			exportName = strings[1]
			importName = strings[2]
		} else if arr, ok := moduleSpecifier.([]string); ok {
			module = arr[0]
			exportName = arr[1]
			importName = arr[1]

		} else if str, ok := moduleSpecifier.(string); ok {
			module = str
			exportName = "default"
			importName = "defaultExport"
		} else {
			continue
		}

		if _, exists := injectsByModule[module]; !exists {
			injectsByModule[module] = []moduleIdentifier{}

			modulePath := prefix + strings.ReplaceAll(module, "/", "-")
			virtualModulePathToSpecifier[modulePath] = module
		}

		injectsByModule[module] = append(injectsByModule[module], moduleIdentifier{
			injectedName: injectName,
			exportName:   exportName,
			importName:   importName,
		})
	}

	for k := range virtualModulePathToSpecifier {
		build.InitialOptions.Inject = append(build.InitialOptions.Inject, k)
	}

	polyfillResolved, err := resolvePolyfills(p.BasePath, polyfill)
	if err != nil {
		panic(err)
	}

	build.InitialOptions.Inject = append(build.InitialOptions.Inject, polyfillResolved...)

	build.OnResolve(api.OnResolveOptions{Filter: unenvVirtualModuleReStr}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		return api.OnResolveResult{
			Path: args.Path,
		}, nil
	})

	build.OnLoad(api.OnLoadOptions{Filter: unenvVirtualModuleReStr}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		module := virtualModulePathToSpecifier[args.Path]
		if module == "" {
			panic(fmt.Sprintf(`Expected %s to be mapped to a module specifier`, args.Path))
		}
		injects := injectsByModule[module]

		if len(injects) == 0 {
			panic(fmt.Sprintf(`Expected %s to have at least one injected module`, module))
		}

		imports := make([]string, len(injects))

		for i, inject := range injects {
			if inject.importName == inject.exportName {
				imports[i] = inject.exportName
				continue
			}

			imports[i] = fmt.Sprintf(`%[1]s as %[2]s`, inject.exportName, inject.importName)
		}

		injectContent := make([]string, len(injects))
		for i, inject := range injects {
			injectContent[i] = fmt.Sprintf(`globalThis.%s = %s;`, inject.injectedName, inject.importName)
		}

		contents := fmt.Sprintf(`
				import { %[1]s } from "%[2]s";
				%[3]s
			`, strings.Join(imports, ", "),
			module,
			strings.Join(injectContent, "\n"))

		return api.OnLoadResult{
			Contents: &contents,
		}, nil
	})

}

func pathResolve(paths ...string) string {
	for i := len(paths) - 1; i >= 0; i-- {
		if filepath.IsAbs(paths[i]) {
			return filepath.Clean(filepath.Join(paths[i:]...))
		}
	}
	abs, _ := filepath.Abs(filepath.Join(paths...))
	return abs
}

func nodeResolve(dir, module string) (*string, error) {
	cmd := exec.Command("node", "-e", fmt.Sprintf(`console.log(require.resolve("%s"))`, module))
	cmd.Dir = dir
	output, err := cmd.Output()

	if err != nil {
		return nil, fmt.Errorf("require.resolve failed: %w", err)
	}

	resolved := strings.TrimSpace(string(output))
	return &resolved, nil
}

func resolvePolyfills(dir string, modules []string) ([]string, error) {
	modulesJSON, _ := json.Marshal(modules)
	script := fmt.Sprintf(`console.log(JSON.stringify(%s.map(m => require.resolve(m))))`, modulesJSON)

	cmd := exec.Command("node", "-e", script)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("require.resolve failed: %w", err)
	}

	var resolved []string
	if err := json.Unmarshal(output, &resolved); err != nil {
		return nil, err
	}
	return resolved, nil
}

type unenvConfig struct {
	Alias    map[string]string `json:"alias"`
	Inject   map[string]any    `json:"inject"` // adjust type as needed
	External []string          `json:"external"`
	Polyfill []string          `json:"polyfill"`
}

func (p *NodeJsHybridPlugin) getUnenvConfig(dir string, compatibilityDate string, compatibilityFlags []string) (*unenvConfig, error) {

	fmt.Println("Found directory " + dir)

	cmd := exec.Command(p.PackageManager, "install", "-D", "unenv@2.0.0-rc.24", "@cloudflare/unenv-preset@latest")
	cmd.Dir = dir
	//if err := cmd.Run(); err != nil {
	//	return nil, fmt.Errorf("failed to install unenv: %w", err)
	//}

	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to install unenv: %w", err)
	}
	fmt.Println(string(b))

	flagsJSON, _ := json.Marshal(compatibilityFlags)

	script := `
        import { defineEnv } from "unenv";
        import { getCloudflarePreset } from "@cloudflare/unenv-preset";

        const { alias, inject, external, polyfill } = defineEnv({
            presets: [
                getCloudflarePreset({
                    compatibilityDate: %q,
                    compatibilityFlags: %s,
                }),
                {
                    alias: {
                        debug: "debug",
                    },
                },
            ],
            npmShims: true,
        }).env;
        console.log(JSON.stringify({ alias, inject, external, polyfill }));
    `
	script = fmt.Sprintf(script, compatibilityDate, flagsJSON)

	cmd = exec.Command("node", "--input-type=module", "-e", script)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("node failed: %s", exitErr.Stderr)
		}
		return nil, err
	}

	var config unenvConfig
	if err := json.Unmarshal(output, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func getNodeJsBuiltinModules() []string {
	cmd := exec.Command("node", "-e", "console.log(require('module').builtinModules)")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var modules []string
	if err := json.Unmarshal(output, &modules); err != nil {
		return nil
	}
	return modules
}
