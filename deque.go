package main

import (
	"encoding/json"
	"errors"
	"flag" // TODO: Replace this with pflag for gnu arg support https://pkg.go.dev/github.com/spf13/pflag
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

type Opt struct {
	QueueFile string
	Backend   string
}

type ArgV struct {
	Operation string
	Queue     string
	Data      string
}

func main() {
	flag.Usage = printUsage
	opt := parseFlags()
	args := parseArgV()

	var queueMap map[string]any

	readQueue(&queueMap, opt) // needs pointer bc nil map
	dq := getDq(queueMap, args)

	switch args.Operation {
	case "shift":
		fmt.Println(shift(&dq))
	case "unshift":
		unshift(&dq, args.Data)
	case "pop":
		fmt.Println(pop(&dq))
	case "push":
		push(&dq, args.Data)
	case "keys":
		keys(queueMap)
	case "list":
		printSlice(&dq)
	case "delete":
		delete(queueMap, args.Queue)
	default:
		fmt.Fprintln(os.Stderr, "Invalid Operation:", "'"+string(args.Operation)+"'",
			"Valid Operators: shift, unshift, pop, push, list, delete")
		os.Exit(1)
	}

	queueMap[args.Queue] = dq
	writeQueueFile(toJson(queueMap), opt.QueueFile)
}

func printUsage() {
	out := flag.CommandLine.Output()

	scriptName := filepath.Base(os.Args[0])

	// Custom usage header & description
	fmt.Fprintf(out, "Usage: %s [options] <operation> <queue> [<data>]\n\n", scriptName)
	fmt.Fprintln(out, "Description:")
	fmt.Fprintln(out, "  Perform basic dequeue (double ended queue) operations from the cli")

	// Automated flags list
	fmt.Fprintln(out, "\nAvailable Flags:")
	flag.PrintDefaults()

	// Custom positional arguments (argv) documentation
	fmt.Fprintln(out, "\nPositional Arguments (argv):")
	fmt.Fprintln(out, "  operation    Operation to preform on queue (required).")
	fmt.Fprintln(out, "  queue        Queue to perform operation on (required).")
	fmt.Fprintln(out, "  data         Data to prepend / append (required for unshift, push).")

	// All operations
	fmt.Fprintln(out, "\nAvailable Operations:")
	fmt.Fprintln(out, "  shift        Remove item from front of queue")
	fmt.Fprintln(out, "  unshift      Add item to front of queue")
	fmt.Fprintln(out, "  pop          Remove item from back of queue")
	fmt.Fprintln(out, "  push         Add item to back of queue")
	fmt.Fprintln(out, "  keys         List all queues")
	fmt.Fprintln(out, "  list         List all values in a queue")
	fmt.Fprintln(out, "  delete       Delete a queue")
}

func parseFlags() Opt {
	var opt Opt

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	queueFile := filepath.Join(homeDir, ".dq.json")

	flag.StringVar(&opt.QueueFile, "file", queueFile, "File to store dequeues in")
	// TODO: Implement other storage backends
	//    flag.StringVar(&opt.Backend, "backend", 'file', "Store backend (file, redis, db)")
	flag.Parse()
	return opt
}

func parseArgV() ArgV {
	// Rest of args after flag.Args parsed.
	argv := flag.Args()

	scriptName := filepath.Base(os.Args[0])
	if len(argv) < 2 {
		fmt.Fprintln(os.Stderr, "Usage:", scriptName, "<operation> <queue> [<data>]")
		os.Exit(1)
	}

	args := ArgV{
		argv[0],
		argv[1],
		"",
	}

	if len(argv) > 2 {
		args.Data = argv[2]
	}

	return args
}

func readQueue(queueMap *map[string]any, opt Opt) {
	jsonBytes := readQueueFile(opt.QueueFile)

	//    fmt.Printf("Json File: %s, Bytes: %s\n", opt.QueueFile, jsonBytes)

	if len(jsonBytes) != 0 {
		if err := json.Unmarshal(jsonBytes, &queueMap); err != nil {
			log.Fatalf("Failed to unpack json: %s", err)
		}
	}
}

// returns blizzard
func getDq(queueMap map[string]any, args ArgV) []any {
	dq, ok := queueMap[args.Queue].([]any)
	if !ok {
		// Error if unshift or pop
		if args.Operation == "shift" || args.Operation == "pop" {
			log.Fatalf("Queue " + args.Queue + " not found")
		}

		// Create it for unshift or push
		queueMap[args.Queue] = []any{}
	}
	dq = queueMap[args.Queue].([]any)
	return dq
}

func readQueueFile(queueFile string) []byte {
	content, err := os.ReadFile(queueFile)
	if err != nil {
		// Create file if it doesn't exist
		if errors.Is(err, os.ErrNotExist) {
			file, createErr := os.OpenFile(queueFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
			if createErr != nil {
				log.Fatalf("Failed to create file: %v\n", createErr)
			}

			file.Close()

			// Data is empty since we just created it
			content = []byte{}
		} else {
			log.Fatalf("Failed to read file: %s", err)
		}
	}
	return content
}

func shift(dq *[]any) string {
	first := (*dq)[0]
	*dq = (*dq)[1:]
	return first.(string)
}

func unshift(dq *[]any, data any) {
	*dq = append([]any{data}, *dq...)
	fmt.Println(dq)
}

func pop(dq *[]any) string {
	last := (*dq)[len(*dq)-1]
	*dq = (*dq)[:len(*dq)-1]
	return last.(string)
}

func push(dq *[]any, data any) {
	*dq = append(*dq, data)
}

func keys(queueMap map[string]any) {
	// make slice with len of map
	keys := make([]any, 0, len(queueMap)) // has to be an any slice for printSlice

	for k := range queueMap {
		keys = append(keys, k)
	}

	// Sort any slice by alpha (assumes all strings, which they come from json keys so they are)
	sort.Slice(keys, func(i, j int) bool {
		// Assert to strings
		return keys[i].(string) < keys[j].(string)
	})
	printSlice(&keys)
}

func printSlice(slice *[]any) {
	for i, elm := range *slice {
		fmt.Printf(elm.(string))
		if i < len(*slice)-1 {
			fmt.Printf(", ")
		}
	}
	fmt.Println()
}

func toJson(q map[string]any) string {
	// Convert map to JSON bytes
	jsonBytes, err := json.Marshal(q)
	if err != nil {
		log.Fatalf("Error marshaling to JSON: %v", err)
	}

	// Convert bytes to string and return it
	jsonString := string(jsonBytes)
	return jsonString
}

func writeQueueFile(json, queueFile string) {
	err := os.WriteFile(queueFile, []byte(json), 0644)
	if err != nil {
		log.Fatal(err)
	}
}
