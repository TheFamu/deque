package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExitErrorError(t *testing.T) {
	inner := errors.New("boom")
	ee := &exitError{code: 4, err: inner}

	if got := ee.Error(); got != "boom" {
		t.Errorf("Error() = %q, want %q", got, "boom")
	}
}

func TestExitErrorUnwrap(t *testing.T) {
	inner := errors.New("boom")
	ee := &exitError{code: 4, err: inner}

	if !errors.Is(ee, inner) {
		t.Error("errors.Is should find the inner error through Unwrap")
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string // substring expected on stderr, "" for none
	}{
		{"exitError uses its code", exitWith(errCode["dequeNotFound"], errors.New("boom")), errCode["dequeNotFound"], "error: boom"},
		{"plain error is 1", errors.New("boom"), 1, "error: boom"},
		{"exits cleanly", nil, 0, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			got := exitCode(tc.err, &buf)

			if got != tc.wantCode {
				t.Errorf("code = %d, want %d", got, tc.wantCode)
			}
			if !strings.Contains(buf.String(), tc.wantMsg) {
				t.Errorf("stderr = %q, want it to contain %q", buf.String(), tc.wantMsg)
			}
		})
	}
}

func TestRun(t *testing.T) {
	// Safety net: if a case forgets -file, run() won't touch real ~/.deque.json.
	t.Setenv("HOME", t.TempDir())

	tests := []struct {
		name         string
		args         []string // CLI args, not including the -file flag (added below)
		createFile   bool     // false means the path won't exist on disk
		fileContent  string   // written to the deque file before run()
		filePath     string   // for testing atomic writes only
		checkContent bool     // should check file afterwards
		wantContent  string   // what should be in file afterwards
		wantCode     int      // 0 = success, 1 = plain error, otherwise an errCode value
	}{
		// Successes
		{
			name:        "debug print",
			args:        []string{"-debug", "pop", "groceries"},
			createFile:  true,
			fileContent: `{"groceries": ["milk", "goober food"]}`,
			filePath:    filepath.Join(t.TempDir(), "deques.json"),
			wantCode:    0,
		},
		{
			name:        "dry",
			args:        []string{"-dry", "pop", "groceries"},
			createFile:  true,
			fileContent: `{"groceries": ["milk", "goober food"]}`,
			filePath:    filepath.Join(t.TempDir(), "deques.json"),
			wantCode:    0,
		},
		{
			name:        "keys",
			args:        []string{"keys"},
			createFile:  true,
			fileContent: `{"groceries": ["milk", "goober food"]}`,
			filePath:    filepath.Join(t.TempDir(), "deques.json"),
			wantCode:    0,
		},
		{
			name:        "list",
			args:        []string{"list", "groceries"},
			createFile:  true,
			fileContent: `{"groceries": ["milk", "goober food"]}`,
			filePath:    filepath.Join(t.TempDir(), "deques.json"),
			wantCode:    0,
		},
		{
			name:         "file change on shift",
			args:         []string{"shift", "groceries"},
			createFile:   true,
			fileContent:  `{"groceries": ["milk", "goober food"]}`,
			filePath:     filepath.Join(t.TempDir(), "deques.json"),
			checkContent: true,
			wantContent:  `{"groceries":["goober food"]}`,
			wantCode:     0,
		},
		{
			name:         "file change on unshift",
			args:         []string{"unshift", "groceries", "eggs"},
			createFile:   true,
			fileContent:  `{"groceries": ["milk", "goober food"]}`,
			filePath:     filepath.Join(t.TempDir(), "deques.json"),
			checkContent: true,
			wantContent:  `{"groceries":["eggs","milk","goober food"]}`,
			wantCode:     0,
		},
		{
			name:         "file change on pop",
			args:         []string{"pop", "groceries"},
			createFile:   true,
			fileContent:  `{"groceries": ["milk", "goober food"]}`,
			filePath:     filepath.Join(t.TempDir(), "deques.json"),
			checkContent: true,
			wantContent:  `{"groceries":["milk"]}`,
			wantCode:     0,
		},
		{
			name:         "file change on push",
			args:         []string{"push", "groceries", "eggs"},
			createFile:   true,
			fileContent:  `{"groceries": ["milk", "goober food"]}`,
			filePath:     filepath.Join(t.TempDir(), "deques.json"),
			checkContent: true,
			wantContent:  `{"groceries":["milk","goober food","eggs"]}`,
			wantCode:     0,
		},
		{
			name:         "file Not change with dry",
			args:         []string{"-dry", "push", "groceries", "eggs"},
			createFile:   true,
			fileContent:  `{"groceries": ["milk", "goober food"]}`,
			filePath:     filepath.Join(t.TempDir(), "deques.json"),
			checkContent: true,
			wantContent:  `{"groceries": ["milk", "goober food"]}`,
			wantCode:     0,
		},

		// Failures
		{
			name:        "bad flags",
			args:        []string{"-notarealflag", "-badflag"},
			createFile:  true,
			fileContent: `{}`,
			filePath:    filepath.Join(t.TempDir(), "deques.json"),
			wantCode:    1,
		},
		{
			name:        "bad args",
			args:        []string{"notarealarg", "badarg"},
			createFile:  true,
			fileContent: `{}`,
			filePath:    filepath.Join(t.TempDir(), "deques.json"),
			wantCode:    1,
		},
		{
			name:        "problem reading json",
			args:        []string{"unshift", "somedeque", "thingy"},
			createFile:  true,
			fileContent: `not json`,
			filePath:    filepath.Join(t.TempDir(), "deques.json"),
			wantCode:    errCode["problemReading"],
		},
		{
			name:        "bad apply", // cases tested more thoroughly in TestApply
			args:        []string{"unshift", "nonexistent"},
			createFile:  true,
			fileContent: `{}`,
			filePath:    filepath.Join(t.TempDir(), "deques.json"),
			wantCode:    1,
		},
		{
			name:        "atomic write fail",
			args:        []string{"unshift", "groceries", "milk"},
			createFile:  false,
			fileContent: `{}`,
			filePath:    filepath.Join("/not/a/real/dir", "deques.json"),
			wantCode:    errCode["atomicWrite"],
		},
		{
			name:        "pop nonexistent deque",
			args:        []string{"pop", "nope"},
			createFile:  true,
			fileContent: `{}`,
			filePath:    filepath.Join(t.TempDir(), "deques.json"),
			wantCode:    errCode["dequeNotFound"],
		},
	}

	for _, tc := range tests {
		silenceOutput(t) // run is noisy so we send it to /dev/null
		t.Run(tc.name, func(t *testing.T) {
			path := tc.filePath
			if tc.createFile {
				if err := os.WriteFile(path, []byte(tc.fileContent), 0644); err != nil {
					t.Fatalf("setup: %v", err)
				}
			}

			// Prepend -file so every case uses its own temp file.
			args := append([]string{"-file", path}, tc.args...)

			err := run(args)

			// Translate the error into the exit status main() would use.
			gotCode := 0
			if err != nil {
				var ee *exitError
				if errors.As(err, &ee) {
					gotCode = ee.code
				} else {
					gotCode = 1
				}
			}

			if gotCode != tc.wantCode {
				t.Errorf("exit code = %d, want %d (err: %v)", gotCode, tc.wantCode, err)
			}

			if tc.checkContent {
				bytes, err := os.ReadFile(tc.filePath)
				if err != nil {
					t.Errorf("cannot read %s: %v)", tc.filePath, err)
				}

				content := string(bytes)
				if content != tc.wantContent {
					t.Errorf("content %s, want %s", content, tc.wantContent)
				}
			}
		})
	}
}

func silenceOutput(t *testing.T) {
	t.Helper()
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = devNull, devNull

	t.Cleanup(func() {
		os.Stdout, os.Stderr = oldOut, oldErr
		devNull.Close()
	})
}

func TestApply(t *testing.T) {
	tests := []struct {
		name      string
		dequeMap  map[string]*deque[any]
		args      argV
		want      any
		mapResult map[string]*deque[any]
		wantErr   bool
	}{
		// Successes
		{
			name: "shift",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs"}},
			},
			args: argV{operation: "shift", dequeName: "groceries"},
			want: "milk",
			mapResult: map[string]*deque[any]{
				"groceries": {items: []any{"eggs"}},
			},
			wantErr: false,
		},
		{
			name: "unshift",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs"}},
			},
			args: argV{operation: "unshift", dequeName: "groceries", data: "goober food"},
			want: nil,
			mapResult: map[string]*deque[any]{
				"groceries": {items: []any{"goober food", "milk", "eggs"}},
			},
			wantErr: false,
		},
		{
			name: "pop",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs"}},
			},
			args: argV{operation: "pop", dequeName: "groceries"},
			want: "eggs",
			mapResult: map[string]*deque[any]{
				"groceries": {items: []any{"milk"}},
			},
			wantErr: false,
		},
		{
			name: "push",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs"}},
			},
			args: argV{operation: "push", dequeName: "groceries", data: "goober food"},
			want: nil,
			mapResult: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs", "goober food"}},
			},
			wantErr: false,
		},
		{
			name: "delete",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs", "goober food"}},
				"todos":     {items: []any{"go grocery shopping", "give goobers smooches"}},
			},
			args: argV{operation: "delete", dequeName: "todos"},
			want: nil,
			mapResult: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs", "goober food"}},
			},
			wantErr: false,
		},
		{
			name: "keys",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs", "goober food"}},
				"todos":     {items: []any{"go grocery shopping", "give goobers smooches"}},
			},
			args: argV{operation: "keys"},
			want: nil,
			mapResult: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs", "goober food"}},
				"todos":     {items: []any{"go grocery shopping", "give goobers smooches"}},
			},
			wantErr: false,
		},
		{
			name: "list",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs", "goober food"}},
			},
			args: argV{operation: "list", dequeName: "groceries"},
			want: nil,
			mapResult: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs", "goober food"}},
			},
			wantErr: false,
		},
		{
			name:     "unshift create deque",
			dequeMap: map[string]*deque[any]{},
			args:     argV{operation: "unshift", dequeName: "groceries", data: "milk"},
			want:     nil,
			mapResult: map[string]*deque[any]{
				"groceries": {items: []any{"milk"}},
			},
			wantErr: false,
		},
		{
			name:     "push create deque",
			dequeMap: map[string]*deque[any]{},
			args:     argV{operation: "push", dequeName: "groceries", data: "milk"},
			want:     nil,
			mapResult: map[string]*deque[any]{
				"groceries": {items: []any{"milk"}},
			},
			wantErr: false,
		},

		// Failures
		{
			name: "shift no deque name",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs"}},
			},
			args:      argV{operation: "shift"},
			want:      nil,
			mapResult: nil,
			wantErr:   true,
		},
		{
			name: "pop no deque name",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs"}},
			},
			args:      argV{operation: "pop"},
			want:      nil,
			mapResult: nil,
			wantErr:   true,
		},
		{
			name: "list no deque name",
			dequeMap: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs"}},
			},
			args:      argV{operation: "list"},
			want:      nil,
			mapResult: nil,
			wantErr:   true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			got, err := apply(tc.dequeMap, tc.args)

			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return // no point comparing options after a failure
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
			if !reflect.DeepEqual(tc.dequeMap, tc.mapResult) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestReadDeques(t *testing.T) {
	tests := []struct {
		name       string
		fileMode   os.FileMode
		createFile bool   // false means the path won't exist on disk
		content    string // file contents, only used if createFile is true
		want       map[string]*deque[any]
		wantErr    bool
	}{
		// Successes
		{
			name:       "valid file with no content",
			fileMode:   0644,
			createFile: true,
			content:    ``,
			want:       map[string]*deque[any]{},
			wantErr:    false,
		},
		{
			name:       "valid file with one deque",
			fileMode:   0644,
			createFile: true,
			content:    `{"groceries": ["milk", "eggs"]}`,
			want: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs"}},
			},
			wantErr: false,
		},
		{
			name:       "valid file with two deques",
			fileMode:   0644,
			createFile: true,
			content:    `{"groceries": ["milk", "eggs"], "todos": ["do work", "???", "profit!"]}`,
			want: map[string]*deque[any]{
				"groceries": {items: []any{"milk", "eggs"}},
				"todos":     {items: []any{"do work", "???", "profit!"}},
			},
			wantErr: false,
		},
		{
			name:       "no file",
			fileMode:   0644,
			createFile: false,
			content:    ``,
			want:       map[string]*deque[any]{},
			wantErr:    false,
		},

		// Errors
		{
			name:       "file unreadable",
			fileMode:   0000,
			createFile: true,
			content:    ``,
			want:       map[string]*deque[any]{},
			wantErr:    true,
		},
		{
			name:       "malformed json",
			fileMode:   0644,
			createFile: true,
			content:    `{invalid json here...}`,
			want:       map[string]*deque[any]{},
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Each subtest gets its own temp dir, deleted automatically.
			path := filepath.Join(t.TempDir(), "deques.json")

			if tc.createFile {
				if err := os.WriteFile(path, []byte(tc.content), tc.fileMode); err != nil {
					t.Fatalf("setup: writing test file: %v", err)
				}
			}

			got, err := readDeques(path)

			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return // nothing meaningful to compare after an error
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestPrintUsage(t *testing.T) {
	// Capture output in a buffer instead of letting it go to stderr.
	var buf bytes.Buffer
	fs := flag.NewFlagSet("deque", flag.ContinueOnError)
	fs.SetOutput(&buf)

	if _, err := parseFlags(fs, nil); err != nil {
		t.Fatal(err)
	}

	printUsage(fs)
	got := buf.String()

	for _, want := range []string{"Usage: deque", "-file", "-debug", "-dry"} {
		if !strings.Contains(got, want) {
			t.Errorf("usage missing %q\n%s", want, got)
		}
	}

	wantContains := []string{
		"Usage:",
		"Description:",
		"Available Flags:",
		"Positional Arguments (argv):",
		"Available Operations:",
		"shift",
		"unshift",
		"pop",
		"push",
		"keys",
		"list",
		"delete",
	}

	for _, want := range wantContains {
		t.Run(want, func(t *testing.T) {
			if !strings.Contains(got, want) {
				t.Errorf("usage output missing %q\nfull output:\n%s", want, got)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	// os.UserHomeDir reads $HOME on Linux. Point it at a temp dir so the
	// test doesn't depend on whoever runs it. t.Setenv restores it afterwards.
	home := t.TempDir()
	t.Setenv("HOME", home)
	defaultFile := filepath.Join(home, ".deque.json")

	tests := []struct {
		name    string
		opts    []string
		want    options
		wantErr bool
	}{
		// Successes
		{"defaults", []string{}, options{dequeFile: defaultFile}, false},
		{"-file with space", []string{"-file", "/tmp/custom.json"}, options{dequeFile: "/tmp/custom.json"}, false},
		{"--file with equals", []string{"--file=/tmp/custom.json"}, options{dequeFile: "/tmp/custom.json"}, false},
		{"file plus positional args", []string{"-file", "/tmp/custom.json", "push", "q", "x"}, options{dequeFile: "/tmp/custom.json"}, false},
		{"debug", []string{"-debug"}, options{dequeFile: defaultFile, debug: true}, false},
		{"dry", []string{"-dry"}, options{dequeFile: defaultFile, dry: true}, false},
		{"all flags", []string{"-debug", "-dry", "-file", "/tmp/custom.json"}, options{dequeFile: "/tmp/custom.json", debug: true, dry: true}, false},

		// Failures
		{"-file missing its value", []string{"-file"}, options{}, true},
		{"unknown flag", []string{"-bogus"}, options{}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Fresh FlagSet per case, so flags never get registered twice.
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard) // silence the usage text it prints on errors

			got, err := parseFlags(fs, tc.opts)

			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return // no point comparing options after a failure
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseArgV(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    argV
		wantErr bool
	}{
		// Test successes: one for each shape of operation
		{"keys needs nothing", []string{"keys"}, argV{operation: "keys"}, false},
		{"list needs deque", []string{"list", "mylist"}, argV{operation: "list", dequeName: "mylist"}, false},
		{"pop needs deque", []string{"pop", "mylist"}, argV{operation: "pop", dequeName: "mylist"}, false},
		{"push needs deque and item", []string{"push", "mylist", "hello"}, argV{operation: "push", dequeName: "mylist", data: "hello"}, false},
		{"unshift needs deque and item", []string{"unshift", "mylist", "hello"}, argV{operation: "unshift", dequeName: "mylist", data: "hello"}, false},

		// Error paths
		{"no args", []string{}, argV{}, true},
		{"invalid operation", []string{"explode", "mylist"}, argV{}, true},
		{"list missing deque", []string{"list"}, argV{}, true},
		{"push missing item", []string{"push", "mylist"}, argV{}, true},
		{"push missing everything", []string{"push"}, argV{}, true},
		{"keys with extra arg", []string{"keys", "oops"}, argV{}, true},
		{"push with extra arg", []string{"push", "mylist", "a", "b"}, argV{}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			got, err := parseArgV(tc.args)

			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
