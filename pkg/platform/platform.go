package platform

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/passes"
	parsepass "github.com/distlanglabs/distlang/pkg/passes/parse"
	"github.com/distlanglabs/distlang/pkg/platform/cloudflare"
)

// Platform identifies a build/deploy environment.
type Platform string

const (
	Goja       Platform = "goja"
	Cloudflare Platform = "cloudflare"
)

// Result captures artifacts for a platform.
type Result struct {
	Platform  Platform
	Artifacts []artifacts.Artifact
}

// BuildAll builds all supported platforms, returning per-platform results and errors.
func BuildAll(filePath string) ([]Result, map[Platform]error) {
	platforms := []Platform{Goja, Cloudflare}
	var results []Result
	errs := make(map[Platform]error)

	for _, p := range platforms {
		res, err := buildPlatform(p, filePath)
		if err != nil {
			errs[p] = err
			continue
		}
		results = append(results, res)
	}
	return results, errs
}

func buildPlatform(p Platform, filePath string) (Result, error) {
	switch p {
	case Goja:
		out, err := passes.Execute(filePath, passes.Options{Format: parsepass.FormatGoja})
		if err != nil {
			return Result{}, err
		}
		art := artifacts.Artifact{Path: filepath.Join("dist", string(Goja), "worker.js"), Content: []byte(out.Emitted)}
		return Result{Platform: Goja, Artifacts: []artifacts.Artifact{art}}, nil
	case Cloudflare:
		out, err := passes.Execute(filePath, passes.Options{Format: parsepass.FormatCloudflare})
		if err != nil {
			return Result{}, err
		}
		fileBase := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		rendered, err := cloudflare.Render(out.Emitted, cloudflare.Context{ProjectName: fileBase})
		if err != nil {
			return Result{}, err
		}
		return Result{Platform: Cloudflare, Artifacts: rendered}, nil
	default:
		return Result{}, fmt.Errorf("unsupported platform: %s", p)
	}
}
