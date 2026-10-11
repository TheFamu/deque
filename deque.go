package main

import (
	"encoding/json"
	"errors"
	"flag" // TODO: Replace this with pflag for gnu arg support https://pkg.go.dev/github.com/spf13/pflag
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var errEmptyDeque = errors.New("deque is empty")

var ops = map[string]opSpec{
	"shift":   {true, false},
	"unshift": {true, true},
	"pop":     {true, false},
	"push":    {true, true},
	"keys":    {false, false},
	"list":    {true, false},
	"delete":  {true, false},
}

// opt is a struct holding flag arguments.
type options struct {
	dequeFile string
	backend   string
	debug     bool
	dry       bool
}

// argV is a struct holding non flag arguments.
type argV struct {
	operation string
	dequeName string
	data      string
}

// opSpec is a struct holding specification for argV args.
type opSpec struct {
	needsDeque bool
	needsItem  bool
}

// Double ended queue data structure holding items of type T.
type deque[T any] struct {
	items []T
}

// UnmarshalJSON lets the json package unpack a standard JSON array directly into our struct.
func (dq *deque[T]) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &dq.items)
}

// MarshalJSON ensures the struct serializes back down to a clean JSON array on disk.
func (dq deque[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(dq.items)
}

// shift removes and returns the front item.
func (dq *deque[T]) shift() (T, error) {
	var zero T
	if len(dq.items) == 0 {
		return zero, errEmptyDeque
	}

	item := dq.items[0]

	// Zero out to avoid keeping references alive in memory.
	dq.items[0] = zero
	dq.items = dq.items[1:]

	return item, nil
}

// unshift add item to front of the deque.
func (dq *deque[T]) unshift(item T) {
	dq.items = append([]T{item}, dq.items...)
}

// pop removes and returns item from the back of the deque.
func (dq *deque[T]) pop() (T, error) {
	var zero T
	n := len(dq.items)
	if n == 0 {
		return zero, errEmptyDeque
	}
	item := dq.items[n-1]
	dq.items[n-1] = zero // avoid memory leak
	dq.items = dq.items[:n-1]
	return item, nil
}

// push adds an item to the back of the deque.
func (dq *deque[T]) push(item T) {
	dq.items = append(dq.items, item)
}

func main() {
	var item any
	var err error
	var dq *deque[any]

	flag.Usage = printUsage

	opt, err := parseFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printUsage()
		os.Exit(1)
	}

	args, err := parseArgV()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printUsage()
		os.Exit(1)
	}

	// Mapping of string keys to deque structs.
	var dequeMap map[string]*deque[any]

	dequeMap, err = readDeques(opt.dequeFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading deques: %s\n", err)
		os.Exit(1)
	}

	// debug after read json, before mutation.
	if opt.debug {
		b, err := json.MarshalIndent(dequeMap, "", "  ")
		if err != nil {
			fmt.Println(os.Stderr, "debug error marshaling map:", err)
		}
		fmt.Fprintf(os.Stderr, "%s\n", string(b))
	}

	dq = dequeMap[args.dequeName]

	switch args.operation {
	case "shift":
		if dq == nil {
			fmt.Fprintf(os.Stderr, "no deque named: '%s'\n", args.dequeName)
			os.Exit(4)
		}
		item, err = dq.shift()
		if err == nil {
			fmt.Println(item)
		}
	case "unshift":
		// Create deque if doesn't exist.
		dq = createDeque(dequeMap, args.dequeName)
		dq.unshift(args.data)
	case "pop":
		if dq == nil {
			fmt.Fprintf(os.Stderr, "no deque named: '%s'\n", args.dequeName)
			os.Exit(4)
		}
		item, err = dq.pop()
		if err == nil {
			fmt.Println(item)
		}
	case "push":
		// Create deque if doesn't exist.
		dq = createDeque(dequeMap, args.dequeName)
		dq.push(args.data)
	case "keys":
		keys(dequeMap)
		os.Exit(0) // Just bail after printing keys, no write
	case "list":
		if dq == nil {
			fmt.Fprintf(os.Stderr, "no deque named: '%s'\n", args.dequeName)
			os.Exit(4)
		}
		printSlice(dq.items)
		os.Exit(0) // Quit after listing so doesn't write every time it lists
	case "delete":
		delete(dequeMap, args.dequeName)
	default:
		panic("unhandled op " + args.operation)
	}

	// If any case errors
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Bail before writing to file.
	if opt.dry {
		os.Exit(0)
	}

	// Update map with new deque (except for on delete)
	if args.operation != "delete" {
		dequeMap[args.dequeName] = dq
	}

	// Convert dequeMap to jsonBytes
	jsonBytes, err := json.Marshal(dequeMap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "problem marshaling dequeMap:", err)
		os.Exit(5)
	}

	// Write jsonBytes to file atomically
	err = writeFileAtomic(opt.dequeFile, jsonBytes, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write json to file: %s\n", err)
		os.Exit(6)
	}
}

func printUsage() {
	out := flag.CommandLine.Output()

	scriptName := filepath.Base(os.Args[0])

	// Custom usage header & description
	fmt.Fprintf(out, "Usage: %s [options] <operation> <deque> [<item>]\n\n", scriptName)
	fmt.Fprintln(out, "Description:")
	fmt.Fprintln(out, "  Perform basic deque (double ended queue) operations from the cli")

	// Available flags list
	fmt.Fprintln(out, "\nAvailable Flags:")
	flag.PrintDefaults()

	// Custom positional arguments (argv) documentation
	fmt.Fprintln(out, "\nPositional Arguments (argv):")
	fmt.Fprintln(out, "  operation    Operation to perform on deque (required).")
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

func parseFlags() (options, error) {
	var opt options

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return opt, fmt.Errorf("Failed to get home directory: %w", err)
	}

	dequeFile := filepath.Join(homeDir, ".deque.json")

	flag.StringVar(&opt.dequeFile, "file", dequeFile, "File to store deques in")
	flag.BoolVar(&opt.debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&opt.dry, "dry", false, "Don't update file after change")
	// TODO: Implement other storage backends
	//    flag.StringVar(&opt.backend, "backend", 'file', "Store backend (file, redis, db)")
	flag.Parse()
	return opt, nil
}

func validOps() string {
	names := make([]string, 0, len(ops))
	for name := range ops {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func parseArgV() (argV, error) {
	argv := flag.Args()
	if len(argv) == 0 {
		return argV{}, errors.New("missing operation")
	}

	name := argv[0]
	spec, ok := ops[name]
	if !ok {
		return argV{}, fmt.Errorf("invalid operation %q (valid: %s)", name, validOps())
	}

	// Count the positional args this operation expects, including the op name.
	want := 1
	if spec.needsDeque {
		want++
	}
	if spec.needsItem {
		want++
	}

	if len(argv) < want {
		return argV{}, fmt.Errorf("%s: missing argument(s), expected %d, got %d", name, want-1, len(argv)-1)
	}
	if len(argv) > want {
		return argV{}, fmt.Errorf("%s: too many arguments, expected %d, got %d", name, want-1, len(argv)-1)
	}

	// Consume the remaining args in order.
	args := argV{operation: name}
	rest := argv[1:]
	if spec.needsDeque {
		args.dequeName, rest = rest[0], rest[1:]
	}
	if spec.needsItem {
		args.data = rest[0]
	}
	return args, nil
}

func readDeques(path string) (map[string]*deque[any], error) {
	dequeMap := make(map[string]*deque[any])
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return dequeMap, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if len(data) == 0 {
		return dequeMap, nil
	}
	if err := json.Unmarshal(data, &dequeMap); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return dequeMap, nil
}

// Create empty deque and put it in the map
func createDeque(dequeMap map[string]*deque[any], dequeName string) *deque[any] {
	dq, ok := dequeMap[dequeName]
	// if deque already exists, just return it.
	if ok && dq != nil {
		return dq
	}
	// Otherwise make new empty deque
	dq = &deque[any]{items: make([]any, 0)}
	dequeMap[dequeName] = dq
	return dq
}

func keys(dequeMap map[string]*deque[any]) {
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
	for _, value := range slice {
		fmt.Println(value)
	}
}

// writeFileAtomic creates a temp file and moves it to prevent data loss on
// simultaneous writes or ctrl+c mid write.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".deque-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
