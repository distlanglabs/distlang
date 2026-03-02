package main

import (
	"flag"
	"fmt"
	"os"
)

func usage() {
	fmt.Println("distlang - portable distributed app framework (POC)")
	fmt.Println()
	fmt.Println("Distlang is a capability-based framework for building portable serverless apps.")
	fmt.Println("Phase 0 focuses on local development with http, kv, and log capabilities.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  distlang [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h, --help   Show help for distlang")
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if len(os.Args) == 1 {
		fmt.Println("distlang - portable distributed app framework (POC)")
		fmt.Println("Run `distlang -h` for usage.")
	}
}
