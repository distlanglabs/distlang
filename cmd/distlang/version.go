package main

import "fmt"

var version = "dev"
var commit = "unknown"

func versionString() string {
	if commit == "" || commit == "unknown" {
		return fmt.Sprintf("distlang %s", version)
	}
	return fmt.Sprintf("distlang %s (%s)", version, commit)
}

func printVersion() {
	fmt.Println(versionString())
}
