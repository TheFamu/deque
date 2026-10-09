package main

import (
	"encoding/json"
	"errors"
	"flag" // TODO: Replace this with pflag for gnu arg support https://pkg.go.dev/github.com/spf13/pflag
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"log"
	"os"
	"path/filepath"
	"sort"
)

var ErrEmptyDeque = errors.New("deque is empty")

// ArgV is a struct holding flag arguments.
type Opt struct {
	DequeFile string
	Backend   string
	Debug     bool
	Dry       bool
}

// ArgV is a struct holding non flag arguments.
type ArgV struct {
	Operation string
	Deque     string
	Data      string
}

// Double ended queue data structure holding items of type T.
type Deque[T any] struct {
	items []T
}

// UnmarshalJSON lets the json package unpack a standard JSON array directly into our struct.
func (dq *Deque[T]) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &dq.items)
}

// MarshalJSON ensures the struct serializes back down to a clean JSON array on disk.
func (dq Deque[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(dq.items)
}

// Shift removes and returns the front item.
func (dq *Deque[T]) Shift() (T, error) {
	var zero T
	if len(dq.items) == 0 {
		return zero, ErrEmptyDeque
	}

	item := dq.items[0]

	// Zero out to avoid keeping references alive in memory.
	dq.items[0] = zero
	dq.items = dq.items[1:]

	return item, nil
}

// Unshift add item to front of the deque.
func (dq *Deque[T]) Unshift(item T) {
	dq.items = append([]T{item}, dq.items...)
}

// Pop removes and returns item from the back of the deque.
func (dq *Deque[T]) Pop() (T, error) {
	var zero T
	n := len(dq.items)
	if n == 0 {
		return zero, ErrEmptyDeque
	}
	item := dq.items[n-1]
	dq.items[n-1] = zero // avoid memory leak
	dq.items = dq.items[:n-1]
	return item, nil
}

// Push adds an item to the back of the deque.
func (dq *Deque[T]) Push(item T) {
	dq.items = append(dq.items, item)
}

/* Thought I needed this, turns out, I do not...
// Len returns the current size of the deque.
func (dq *Deque[T]) Len() int {
	return len(dq.items)
}

// IsEmpty checks if the deque has no elements.
func (q *Deque[T]) IsEmpty() bool {
	return len(q.items) == 0
}
*/

func main() {
	flag.Usage = printUsage
	opt := parseFlags()
	args := parseArgV()

	var item any
	var err error
	var deque *Deque[any]

	// Mapping of string keys to Deque structs.
	dequeMap := make(map[string]*Deque[any])

	// Populate dequeMap with structs directly from json.
	err = populateDeque(dequeMap, opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to populate dequeMap struct from json: %v", err)
		os.Exit(3)
	}

	// Debug after read, before mutation.
	if opt.Debug {
		spew.Dump(dequeMap)
	}

	// Get the deque struct we're dealing with.
	deque, err = getDeque(dequeMap, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch deque struct from map: %v", err)
		os.Exit(4)
	}

	switch args.Operation {
	case "shift":
		item, err = deque.Shift()
		fmt.Println(item)
	case "unshift":
		if args.Data == "" {
			fmt.Fprintf(os.Stderr, "Error missing unshift item")
			os.Exit(2)
		}
		deque.Unshift(args.Data)
	case "pop":
		item, err = deque.Pop()
		fmt.Println(item)
	case "push":
		if args.Data == "" {
			fmt.Fprintf(os.Stderr, "Error missing unshift item")
			os.Exit(2)
		}
		deque.Push(args.Data)
	case "keys":
		keys(dequeMap)
		os.Exit(0) // Just bail after printing keys, no write
	case "list":
		printSlice(deque.items)
	case "delete":
		delete(dequeMap, args.Deque)
	default:
		fmt.Fprintln(os.Stderr, "Invalid Operation:", "'"+string(args.Operation)+"'",
			"Valid Operators: shift, unshift, pop, push, list, delete")
		printUsage()
		os.Exit(1)
	}

	// If any case errors
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Bail before writing to file.
	if opt.Dry {
		os.Exit(0)
	}

	dequeMap[args.Deque] = deque
	writeDequeFile(toJson(dequeMap), opt.DequeFile)
}

func printUsage() {
	out := flag.CommandLine.Output()

	scriptName := filepath.Base(os.Args[0])

	// Custom usage header & description
	fmt.Fprintf(out, "Usage: %s [options] <operation> <deque> [<item>]\n\n", scriptName)
	fmt.Fprintln(out, "Description:")
	fmt.Fprintln(out, "  Perform basic deque (double ended queue) operations from the cli")

	// Automated flags list
	fmt.Fprintln(out, "\nAvailable Flags:")
	flag.PrintDefaults()

	// Custom positional arguments (argv) documentation
	fmt.Fprintln(out, "\nPositional Arguments (argv):")
	fmt.Fprintln(out, "  operation    Operation to preform on deque (required).")
	fmt.Fprintln(out, "  deque        Deque to perform operation on (required, except keys).")
	fmt.Fprintln(out, "  item         Item to prepend / append (required for unshift, push).")

	// All operations
	fmt.Fprintln(out, "\nAvailable Operations:")
	fmt.Fprintln(out, "  shift        Remove item from front of deque")
	fmt.Fprintln(out, "  unshift      Add item to front of deque")
	fmt.Fprintln(out, "  pop          Remove item from back of deque")
	fmt.Fprintln(out, "  push         Add item to back of deque")
	fmt.Fprintln(out, "  keys         List all deques")
	fmt.Fprintln(out, "  list         List all values in a deque")
	fmt.Fprintln(out, "  delete       Delete a deque")
}

func parseFlags() Opt {
	var opt Opt

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	dequeFile := filepath.Join(homeDir, ".deque.json")

	flag.StringVar(&opt.DequeFile, "file", dequeFile, "File to store dedeques in")
	flag.BoolVar(&opt.Debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&opt.Dry, "dry", false, "Don't update file after change")
	// TODO: Implement other storage backends
	//    flag.StringVar(&opt.Backend, "backend", 'file', "Store backend (file, redis, db)")
	flag.Parse()
	return opt
}

func parseArgV() ArgV {
	// Rest of args after flag.Args parsed.
	argv := flag.Args()
	args := ArgV{}

	switch len(argv) {
	case 0:
		printUsage()
		os.Exit(1)
	case 1:
		args.Operation = argv[0]
	case 2:
		args.Operation = argv[0]
		args.Deque = argv[1]
	case 3:
		args.Operation = argv[0]
		args.Deque = argv[1]
		args.Data = argv[2]
	}

	return args
}

func populateDeque(dequeMap map[string]*Deque[any], opt Opt) error {
	jsonBytes, err := readDequeMapFile(opt.DequeFile)
	if err != nil {
		return err
	}

	//    fmt.Printf("Json File: %s, Bytes: %s\n", opt.DequeFile, jsonBytes)

	if len(jsonBytes) == 0 {
		return fmt.Errorf("Failed to unpack json: %s", err)
	}

	if err := json.Unmarshal(jsonBytes, &dequeMap); err != nil {
		return fmt.Errorf("Failed to unpack json: %s", err)
	}

	return nil
}

// gets dq blizzard
func getDeque(dequeMap map[string]*Deque[any], args ArgV) (*Deque[any], error) {
	dq, ok := dequeMap[args.Deque]
	if !ok {
		// Error if unshift or pop
		if args.Operation == "shift" || args.Operation == "pop" {
			log.Fatalf("Deque " + args.Deque + " not found")
		}

		// Create it for unshift or push
		dq = &Deque[any]{items: make([]any, 0)}
		dequeMap[args.Deque] = dq
	}
	dq = dequeMap[args.Deque]
	return dq, nil
}

func readDequeMapFile(dequeFile string) ([]byte, error) {
	content, err := os.ReadFile(dequeFile)
	if err != nil {
		return content, fmt.Errorf("Failed to read file: %s", err)
	}
	return content, nil
}

func keys(dequeMap map[string]*Deque[any]) {
	// make slice with len of map
	keys := make([]any, 0, len(dequeMap)) // has to be an any slice for printSlice

	for k := range dequeMap {
		keys = append(keys, k)
	}

	// Sort any slice by alpha (assumes all strings, which they come from json keys so they are)
	sort.Slice(keys, func(i, j int) bool {
		// Assert to strings
		return keys[i].(string) < keys[j].(string)
	})
	printSlice(keys)
}

func printSlice(slice []any) {
	for i, item := range slice {
		fmt.Printf(item.(string))
		if i < len(slice)-1 {
			fmt.Printf(", ")
		}
	}
	fmt.Println()
}

func toJson(q map[string]*Deque[any]) string {
	// Convert map to JSON bytes
	jsonBytes, err := json.Marshal(q)
	if err != nil {
		log.Fatalf("Error marshaling to JSON: %v", err)
	}

	// Convert bytes to string and return it
	jsonString := string(jsonBytes)
	return jsonString
}

func writeDequeFile(json, dequeFile string) {
	err := os.WriteFile(dequeFile, []byte(json), 0644)
	if err != nil {
		log.Fatal(err)
	}
}
