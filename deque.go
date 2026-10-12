package main

import (
	"encoding/json"
	"errors"
	"flag" // TODO: Replace this with pflag for gnu arg support https://pkg.go.dev/github.com/spf13/pflag
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ops = map[string]opSpec{
	"shift":   {true, false},
	"unshift": {true, true},
	"pop":     {true, false},
	"push":    {true, true},
	"keys":    {false, false},
	"list":    {true, false},
	"delete":  {true, false},
}

// errCode is a map containing named error codes for critical error exit statuses.
var errCode = map[string]int{
	"problemReading": 3,
	"dequeNotFound":  4,
	"marshingJson":   5,
	"atomicWrite":    6,
}

var errEmptyDeque = errors.New("deque is empty")

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

// exitError is a struct holding error message and exit code.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

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

// main does almost nothing except calls exitCode to print errors and call
// os.Exit(). The real work happens in run().
func main() {
	os.Exit(exitCode(run(os.Args[1:]), os.Stderr))
}

// exitCode maps an error from run() to a process exit status,
// printing the error message to stderr along the way.
func exitCode(err error, stderr io.Writer) int {
	if err == nil || errors.Is(err, flag.ErrHelp) {
		return 0
	}

	var ee *exitError
	if errors.As(err, &ee) {
		fmt.Fprintln(stderr, "error:", ee.err)
		return ee.code
	}

	fmt.Fprintln(stderr, "error:", err)
	return 1
}

// run houses main application logic. Returns only errors.
func run(cliArgs []string) error {
	fs := flag.NewFlagSet("deque", flag.ContinueOnError)
	fs.Usage = func() { printUsage(fs) }

	opts, err := parseFlags(fs, cliArgs)
	if err != nil {
		return err
	}

	args, err := parseArgV(fs.Args())
	if err != nil {
		printUsage(fs)
		return err
	}
	dequeMap, err := readDeques(opts.dequeFile)
	if err != nil {
		return exitWith(errCode["problemReading"], fmt.Errorf("reading deques: %w", err))
	}

	if opts.debug {
		debugDump(dequeMap)
	}

	out, err := apply(dequeMap, args)
	if err != nil {
		return err
	}
	if opts.dry || args.operation == "keys" || args.operation == "list" {
		return nil
	}

	if opts.debug {
		debugDump(dequeMap)
	}

	jsonBytes, err := json.Marshal(dequeMap)
	if err != nil {
		return exitWith(errCode["marshingJson"], fmt.Errorf("marshaling deques: %w", err))
	}
	if err := writeFileAtomic(opts.dequeFile, jsonBytes, 0644); err != nil {
		return exitWith(errCode["atomicWrite"], fmt.Errorf("writing %s: %w", opts.dequeFile, err))
	}
	if out != nil {
		fmt.Println(out) // print only after a successful write
	}
	return nil
}

// exitWith is a wrapper for returning exitError with error message and exit code.
func exitWith(code int, err error) error {
	return &exitError{code: code, err: err}
}

// apply house the main switch statement logic for handling deque operations.
func apply(dequeMap map[string]*deque[any], args argV) (any, error) {
	dq := dequeMap[args.dequeName]
	needExisting := func() error {
		if dq == nil {
			return exitWith(errCode["dequeNotFound"], fmt.Errorf("no deque named %q", args.dequeName))
		}
		return nil
	}

	switch args.operation {
	case "shift":
		if err := needExisting(); err != nil {
			return nil, err
		}
		return dq.shift()
	case "pop":
		if err := needExisting(); err != nil {
			return nil, err
		}
		return dq.pop()
	case "push":
		createDeque(dequeMap, args.dequeName).push(args.data)
	case "unshift":
		createDeque(dequeMap, args.dequeName).unshift(args.data)
	case "delete":
		delete(dequeMap, args.dequeName)
	case "keys":
		keys(dequeMap)
	case "list":
		if err := needExisting(); err != nil {
			return nil, err
		}
		printSlice(dq.items)
	}
	return nil, nil
}

func printUsage(fs *flag.FlagSet) {
	out := fs.Output()

	scriptName := filepath.Base(os.Args[0])

	// Custom usage header & description
	fmt.Fprintf(out, "Usage: %s [options] <operation> <deque> [<item>]\n\n", scriptName)
	fmt.Fprintln(out, "Description:")
	fmt.Fprintln(out, "  Perform basic deque (double ended queue) operations from the cli")

	// Available flags list
	fmt.Fprintln(out, "\nAvailable Flags:")
	fs.PrintDefaults()

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

func parseFlags(fs *flag.FlagSet, args []string) (options, error) {
	var opt options

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return opt, fmt.Errorf("Failed to get home directory: %w", err)
	}

	dequeFile := filepath.Join(homeDir, ".deque.json")

	fs.StringVar(&opt.dequeFile, "file", dequeFile, "File to store deques in")
	fs.BoolVar(&opt.debug, "debug", false, "Enable debug mode")
	fs.BoolVar(&opt.dry, "dry", false, "Don't update file after change")
	// TODO: Implement other storage backends
	//    flag.StringVar(&opt.backend, "backend", 'file', "Store backend (file, redis, db)")

	if err := fs.Parse(args); err != nil {
		return opt, err
	}

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

func parseArgV(argv []string) (argV, error) {
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

// readDeques creates a new dequeMap and populates it by unmarshaling json from
// dequeMap file, if file exists and has contents. Otherwise returns empty map
// to be populated later by operations.
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

// debugDump is helper that prints map as json for diagnostic purposes.
func debugDump(m map[string]*deque[any]) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "debug error marshaling map:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "%s\n", b)
}

// createDeque creates empty deque and put it in the map.
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
	names := make([]any, 0, len(dequeMap)) // has to be an any slice for printSlice

	for k := range dequeMap {
		names = append(names, k)
	}

	// Sort any slice by alpha (assumes all strings, which they come from json keys so they are)
	sort.Slice(names, func(i, j int) bool {
		// Assert to strings
		return names[i].(string) < names[j].(string)
	})
	printSlice(names)
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
