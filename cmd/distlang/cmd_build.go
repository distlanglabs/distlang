package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/backend"
	cloudflareprovider "github.com/distlanglabs/distlang/pkg/provider/cloudflare"
)

func runBuild(args []string) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpBuild()
		return 0
	}

	filePath, err := singlePathArg(args, "build")
	if err != nil {
		return 1
	}

	v8Out, err := backend.BuildV8(filePath)
	hasError := false
	if err != nil {
		fmt.Fprintf(os.Stderr, "v8: build backend artifacts: %v\n", err)
		hasError = true
	} else {
		if err := artifacts.WriteAll(v8Out.Artifacts); err != nil {
			fmt.Fprintf(os.Stderr, "v8: write artifacts: %v\n", err)
			hasError = true
		}

		ctx, err := cloudflareBuildContext(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cloudflare: load target config: %v\n", err)
			hasError = true
		} else {
			packaged, err := cloudflareprovider.Package(v8Out, ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cloudflare: package provider artifacts: %v\n", err)
				hasError = true
			} else if err := artifacts.WriteAll(packaged); err != nil {
				fmt.Fprintf(os.Stderr, "cloudflare: write artifacts: %v\n", err)
				hasError = true
			}
		}
	}

	if hasError {
		fmt.Fprintln(os.Stderr, "Build failed")
		return 1
	}

	fmt.Println("Build succeeded")
	if len(v8Out.Workers) > 0 {
		fmt.Println("- v8: dist/v8/handlerSet1/worker.js, dist/v8/handlerSet2/worker.js")
		fmt.Println("- cloudflare: dist/cloudflare/handlerSet1/*, dist/cloudflare/handlerSet2/*")
	} else {
		fmt.Println("- v8: dist/v8/worker.js")
		fmt.Println("- cloudflare: dist/cloudflare/worker.js, dist/cloudflare/wrangler.toml, dist/cloudflare/Makefile")
	}
	fmt.Println("- generated helpers: generated/distlang/* (when distlang, distlang/core, or distlang/layers is imported)")

	return 0
}

func cloudflareBuildContext(filePath string) (cloudflareprovider.Context, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return cloudflareprovider.Context{}, err
	}

	ctx := cloudflareprovider.Context{
		ProjectName:   fileBase(filePath),
		KVBindingName: "DISTLANG_KV",
	}

	envPath := filepath.Join(filepath.Dir(absFilePath), "targets", "cloudflare", "cloudflare.env")
	values, err := loadEnvFile(envPath)
	if err != nil {
		return cloudflareprovider.Context{}, err
	}

	ctx.KVNamespaceID = strings.TrimSpace(values["CLOUDFLARE_KV_NAMESPACE_ID"])
	ctx.KVPreviewID = strings.TrimSpace(values["CLOUDFLARE_KV_PREVIEW_ID"])
	ctx.StoreBaseURL = strings.TrimSpace(values["DISTLANG_STORE_BASE_URL"])
	ctx.HelpersMode = strings.TrimSpace(values["DISTLANG_HELPERS_MODE"])

	return ctx, nil
}

func fileBase(filePath string) string {
	base := filepath.Base(filePath)
	return base[:len(base)-len(filepath.Ext(base))]
}
