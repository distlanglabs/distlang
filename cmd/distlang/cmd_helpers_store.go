package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/distlanglabs/distlang/pkg/auth"
	storeapi "github.com/distlanglabs/distlang/pkg/store"
)

func runHelpersStore(args []string) int {
	if len(args) == 0 {
		commandHelpHelpersStore()
		return 1
	}

	if args[0] == "-h" || args[0] == "--help" {
		commandHelpHelpersStore()
		return 0
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "objectdb":
		return runHelpersStoreObjectDB(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown helpers store subcommand: %s\n", subcommand)
		commandHelpHelpersStore()
		return 1
	}
}

func runHelpersStoreObjectDB(args []string) int {
	if len(args) == 0 {
		commandHelpHelpersStoreObjectDB()
		return 1
	}

	if args[0] == "-h" || args[0] == "--help" {
		commandHelpHelpersStoreObjectDB()
		return 0
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "status":
		return runHelpersStoreObjectDBStatus(commandArgs)
	case "buckets":
		return runHelpersStoreObjectDBBuckets(commandArgs)
	case "keys":
		return runHelpersStoreObjectDBKeys(commandArgs)
	case "put":
		return runHelpersStoreObjectDBPut(commandArgs)
	case "get":
		return runHelpersStoreObjectDBGet(commandArgs)
	case "head":
		return runHelpersStoreObjectDBHead(commandArgs)
	case "delete":
		return runHelpersStoreObjectDBDelete(commandArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown helpers store objectdb subcommand: %s\n", command)
		commandHelpHelpersStoreObjectDB()
		return 1
	}
}

func runHelpersStoreObjectDBStatus(args []string) int {
	if len(args) > 0 {
		if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
			commandHelpHelpersStoreObjectDBStatus()
			return 0
		}
		fmt.Fprintln(os.Stderr, "helpers store objectdb status does not accept positional arguments")
		return 1
	}

	client, session, err := newObjectDBClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb status failed: %v\n", err)
		return 1
	}

	status, err := client.ObjectDBStatus(session.AccessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb status failed: %v\n", err)
		return 1
	}

	if status.User.Name != "" {
		fmt.Printf("User: %s <%s>\n", status.User.Name, status.User.Email)
	} else {
		fmt.Printf("User: %s\n", status.User.Email)
	}
	fmt.Printf("Service: %s %s\n", status.Service, status.Version)
	fmt.Printf("Store base URL: %s\n", client.BaseURL())
	fmt.Printf("Buckets route: %s\n", status.Routes.Buckets)
	fmt.Printf("Values route: %s\n", status.Routes.Values)
	fmt.Printf("Keys route: %s\n", status.Routes.Keys)
	return 0
}

func runHelpersStoreObjectDBBuckets(args []string) int {
	if len(args) == 0 {
		commandHelpHelpersStoreObjectDBBuckets()
		return 1
	}
	if args[0] == "-h" || args[0] == "--help" {
		commandHelpHelpersStoreObjectDBBuckets()
		return 0
	}

	command := args[0]
	commandArgs := args[1:]

	client, session, err := newObjectDBClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb buckets %s failed: %v\n", command, err)
		return 1
	}

	switch command {
	case "list":
		if len(commandArgs) > 0 {
			fmt.Fprintln(os.Stderr, "helpers store objectdb buckets list does not accept positional arguments")
			return 1
		}
		response, err := client.ListBuckets(session.AccessToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "helpers store objectdb buckets list failed: %v\n", err)
			return 1
		}
		if len(response.Buckets) == 0 {
			fmt.Println("No buckets")
			return 0
		}
		for _, bucket := range response.Buckets {
			if strings.TrimSpace(bucket.CreatedAt) != "" {
				fmt.Printf("%s\t%s\n", bucket.Name, bucket.CreatedAt)
			} else {
				fmt.Println(bucket.Name)
			}
		}
		return 0
	case "create":
		bucket, ok := singleValueArg(commandArgs, "helpers store objectdb buckets create <bucket>")
		if !ok {
			return 1
		}
		response, err := client.CreateBucket(session.AccessToken, bucket)
		if err != nil {
			fmt.Fprintf(os.Stderr, "helpers store objectdb buckets create failed: %v\n", err)
			return 1
		}
		fmt.Printf("Created bucket %s\n", response.Bucket)
		return 0
	case "exists":
		bucket, ok := singleValueArg(commandArgs, "helpers store objectdb buckets exists <bucket>")
		if !ok {
			return 1
		}
		exists, err := client.BucketExists(session.AccessToken, bucket)
		if err != nil {
			fmt.Fprintf(os.Stderr, "helpers store objectdb buckets exists failed: %v\n", err)
			return 1
		}
		if exists {
			fmt.Printf("Bucket %s exists\n", bucket)
		} else {
			fmt.Printf("Bucket %s does not exist\n", bucket)
		}
		return 0
	case "delete":
		bucket, ok := singleValueArg(commandArgs, "helpers store objectdb buckets delete <bucket>")
		if !ok {
			return 1
		}
		response, err := client.DeleteBucket(session.AccessToken, bucket)
		if err != nil {
			fmt.Fprintf(os.Stderr, "helpers store objectdb buckets delete failed: %v\n", err)
			return 1
		}
		fmt.Printf("Deleted bucket %s\n", response.Bucket)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown helpers store objectdb buckets subcommand: %s\n", command)
		commandHelpHelpersStoreObjectDBBuckets()
		return 1
	}
}

func runHelpersStoreObjectDBKeys(args []string) int {
	if len(args) == 0 {
		commandHelpHelpersStoreObjectDBKeys()
		return 1
	}
	if args[0] == "-h" || args[0] == "--help" {
		commandHelpHelpersStoreObjectDBKeys()
		return 0
	}

	if args[0] != "list" {
		fmt.Fprintf(os.Stderr, "unknown helpers store objectdb keys subcommand: %s\n", args[0])
		commandHelpHelpersStoreObjectDBKeys()
		return 1
	}

	bucket := ""
	opts := storeapi.ListKeysOptions{}
	for _, arg := range args[1:] {
		switch {
		case strings.HasPrefix(arg, "--prefix="):
			opts.Prefix = strings.TrimPrefix(arg, "--prefix=")
		case strings.HasPrefix(arg, "--limit="):
			limitValue := strings.TrimPrefix(arg, "--limit=")
			limit, err := strconv.Atoi(limitValue)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid limit: %s\n", limitValue)
				return 1
			}
			opts.Limit = limit
		case strings.HasPrefix(arg, "--cursor="):
			opts.Cursor = strings.TrimPrefix(arg, "--cursor=")
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", arg)
			return 1
		case bucket == "":
			bucket = arg
		default:
			fmt.Fprintln(os.Stderr, "helpers store objectdb keys list accepts only one bucket")
			return 1
		}
	}

	if bucket == "" {
		fmt.Fprintln(os.Stderr, "helpers store objectdb keys list requires a bucket")
		return 1
	}

	client, session, err := newObjectDBClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb keys list failed: %v\n", err)
		return 1
	}

	response, err := client.ListKeys(session.AccessToken, bucket, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb keys list failed: %v\n", err)
		return 1
	}

	if len(response.Keys) == 0 {
		fmt.Printf("No keys in bucket %s\n", bucket)
	} else {
		for _, key := range response.Keys {
			if key.Metadata != nil {
				fmt.Printf("%s\t%s\t%d\t%s\n", key.Name, key.Metadata.ContentType, key.Metadata.Size, key.Metadata.UpdatedAt)
			} else {
				fmt.Println(key.Name)
			}
		}
	}

	fmt.Printf("List complete: %t\n", response.ListComplete)
	if strings.TrimSpace(response.Cursor) != "" {
		fmt.Printf("Cursor: %s\n", response.Cursor)
	}
	return 0
}

func runHelpersStoreObjectDBPut(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		commandHelpHelpersStoreObjectDBPut()
		return 0
	}

	bucket, key, options, ok := parsePutArgs(args)
	if !ok {
		return 1
	}

	body, contentType, err := resolvePutInput(options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb put failed: %v\n", err)
		return 1
	}

	client, session, err := newObjectDBClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb put failed: %v\n", err)
		return 1
	}

	response, err := client.PutValue(session.AccessToken, bucket, key, body, contentType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb put failed: %v\n", err)
		return 1
	}

	fmt.Printf("Wrote %s/%s\n", response.Bucket, response.Key)
	fmt.Printf("Content-Type: %s\n", response.Metadata.ContentType)
	fmt.Printf("Size: %d\n", response.Metadata.Size)
	fmt.Printf("Updated: %s\n", response.Metadata.UpdatedAt)
	return 0
}

func runHelpersStoreObjectDBGet(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		commandHelpHelpersStoreObjectDBGet()
		return 0
	}

	bucket := ""
	key := ""
	responseType := "text"
	outputPath := ""
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--type="):
			responseType = strings.TrimPrefix(arg, "--type=")
		case strings.HasPrefix(arg, "--output="):
			outputPath = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", arg)
			return 1
		case bucket == "":
			bucket = arg
		case key == "":
			key = arg
		default:
			fmt.Fprintln(os.Stderr, "helpers store objectdb get accepts only one bucket and key")
			return 1
		}
	}

	if bucket == "" || key == "" {
		fmt.Fprintln(os.Stderr, "Usage: distlang helpers store objectdb get <bucket> <key> [--type=text|json|bytes] [--output=path]")
		return 1
	}
	if responseType != "text" && responseType != "json" && responseType != "bytes" {
		fmt.Fprintf(os.Stderr, "invalid type: %s\n", responseType)
		return 1
	}
	if responseType == "bytes" && strings.TrimSpace(outputPath) == "" {
		fmt.Fprintln(os.Stderr, "helpers store objectdb get requires --output=path when --type=bytes")
		return 1
	}

	client, session, err := newObjectDBClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb get failed: %v\n", err)
		return 1
	}

	response, err := client.GetValue(session.AccessToken, bucket, key, responseType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb get failed: %v\n", err)
		return 1
	}

	if outputPath != "" {
		if err := writeOutputFile(outputPath, response.Body); err != nil {
			fmt.Fprintf(os.Stderr, "helpers store objectdb get failed: %v\n", err)
			return 1
		}
		fmt.Printf("Wrote %s/%s to %s\n", bucket, key, outputPath)
	} else {
		switch responseType {
		case "json":
			var value any
			if err := json.Unmarshal(response.Body, &value); err != nil {
				fmt.Fprintf(os.Stderr, "helpers store objectdb get failed: decode json: %v\n", err)
				return 1
			}
			pretty, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "helpers store objectdb get failed: format json: %v\n", err)
				return 1
			}
			fmt.Println(string(pretty))
		default:
			fmt.Print(string(response.Body))
			if len(response.Body) == 0 || response.Body[len(response.Body)-1] != '\n' {
				fmt.Println()
			}
		}
	}

	fmt.Printf("Content-Type: %s\n", response.ContentType)
	if response.Size != "" {
		fmt.Printf("Size: %s\n", response.Size)
	}
	if response.UpdatedAt != "" {
		fmt.Printf("Updated: %s\n", response.UpdatedAt)
	}
	return 0
}

func runHelpersStoreObjectDBHead(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		commandHelpHelpersStoreObjectDBHead()
		return 0
	}

	bucket, key, ok := doubleValueArg(args, "helpers store objectdb head <bucket> <key>")
	if !ok {
		return 1
	}

	client, session, err := newObjectDBClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb head failed: %v\n", err)
		return 1
	}

	response, err := client.HeadValue(session.AccessToken, bucket, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb head failed: %v\n", err)
		return 1
	}

	fmt.Printf("Key: %s/%s\n", bucket, key)
	fmt.Printf("Content-Type: %s\n", response.ContentType)
	fmt.Printf("Content-Length: %s\n", response.ContentSize)
	if response.UpdatedAt != "" {
		fmt.Printf("Updated: %s\n", response.UpdatedAt)
	}
	return 0
}

func runHelpersStoreObjectDBDelete(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		commandHelpHelpersStoreObjectDBDelete()
		return 0
	}

	bucket, key, ok := doubleValueArg(args, "helpers store objectdb delete <bucket> <key>")
	if !ok {
		return 1
	}

	client, session, err := newObjectDBClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb delete failed: %v\n", err)
		return 1
	}

	response, err := client.DeleteValue(session.AccessToken, bucket, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers store objectdb delete failed: %v\n", err)
		return 1
	}

	fmt.Printf("Deleted %s/%s\n", response.Bucket, response.Key)
	return 0
}

type putOptions struct {
	filePath    string
	text        string
	hasText     bool
	contentType string
}

func parsePutArgs(args []string) (string, string, putOptions, bool) {
	bucket := ""
	key := ""
	options := putOptions{}
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--file="):
			options.filePath = strings.TrimPrefix(arg, "--file=")
		case strings.HasPrefix(arg, "--text="):
			options.text = strings.TrimPrefix(arg, "--text=")
			options.hasText = true
		case strings.HasPrefix(arg, "--content-type="):
			options.contentType = strings.TrimPrefix(arg, "--content-type=")
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", arg)
			return "", "", putOptions{}, false
		case bucket == "":
			bucket = arg
		case key == "":
			key = arg
		default:
			fmt.Fprintln(os.Stderr, "helpers store objectdb put accepts only one bucket and key")
			return "", "", putOptions{}, false
		}
	}
	if bucket == "" || key == "" {
		fmt.Fprintln(os.Stderr, "Usage: distlang helpers store objectdb put <bucket> <key> [--file=path | --text=value] [--content-type=type]")
		return "", "", putOptions{}, false
	}
	return bucket, key, options, true
}

func resolvePutInput(options putOptions) ([]byte, string, error) {
	hasFile := strings.TrimSpace(options.filePath) != ""
	hasText := options.hasText
	if hasFile == hasText {
		return nil, "", errors.New("choose exactly one of --file=path or --text=value")
	}
	if hasFile {
		body, err := os.ReadFile(options.filePath)
		if err != nil {
			return nil, "", fmt.Errorf("read %s: %w", options.filePath, err)
		}
		contentType := strings.TrimSpace(options.contentType)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		return body, contentType, nil
	}
	contentType := strings.TrimSpace(options.contentType)
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	return []byte(options.text), contentType, nil
}

func newObjectDBClient() (*storeapi.Client, auth.Session, error) {
	authClient := auth.NewClient(auth.ResolveBaseURL())
	session, err := authClient.EnsureSession()
	if err != nil {
		return nil, auth.Session{}, err
	}
	return storeapi.NewClient(storeapi.ResolveBaseURL()), session, nil
}

func singleValueArg(args []string, usage string) (string, bool) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: distlang %s\n", usage)
		return "", false
	}
	return args[0], true
}

func doubleValueArg(args []string, usage string) (string, string, bool) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: distlang %s\n", usage)
		return "", "", false
	}
	return args[0], args[1], true
}

func writeOutputFile(filePath string, body []byte) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(absPath, body, 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}
	return nil
}
