package plugins

import (
	"path/filepath"
	"slices"

	"github.com/evanw/esbuild/pkg/api"
)

type ExternalFilePlugin struct {
	Extensions []string
}

func (p *ExternalFilePlugin) New() api.Plugin {
	return api.Plugin{
		Name: "external-files",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				ext := filepath.Ext(args.Path)

				if slices.Contains(p.Extensions, ext) {
					return api.OnResolveResult{
						Path:     args.Path,
						External: true,
					}, nil
				}

				return api.OnResolveResult{}, nil
			})
		},
	}
}
