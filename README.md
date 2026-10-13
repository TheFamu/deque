# Deque

[![Test](https://github.com/TheFamu/deque/actions/workflows/actions.yml/badge.svg?branch=main)](https://github.com/TheFamu/deque/actions/workflows/actions.yml)

A double ended queue cli utility written in Go.

## Installation

Download the archive for your platform from the [latest release](https://github.com/TheFamu/deque/releases/latest), 
extract it, and put the `deque` binary somewhere on your `PATH`.

### Linux / macOS

Easy install via bash script:

```bash
curl https://raw.githubusercontent.com/TheFamu/deque/refs/heads/main/install.sh | bash
```

Manual Install:

```shell
VERSION=v0.1.0
OS=linux      # or: darwin
ARCH=amd64    # or: arm64

curl -LO "https://github.com/TheFamu/deque/releases/download/${VERSION}/deque_${VERSION}_${OS}_${ARCH}.tar.gz"
tar -xzf "deque_${VERSION}_${OS}_${ARCH}.tar.gz"
sudo install -m 755 "deque_${VERSION}_${OS}_${ARCH}/deque" /usr/local/bin/deque
```

### Windows

Download `deque_<version>_windows_amd64.zip` (or `arm64`) from the releases page, extract it, and add the folder containing `deque.exe` to your `PATH`.

### Verify

Each release includes a `checksums.txt` file. To check your download:

```shell
# Linux
sha256sum -c checksums.txt --ignore-missing

# macOS
shasum -a 256 -c checksums.txt --ignore-missing
```

Then run `deque --help` to confirm it works.

## Usage

```
Usage: deque [options] <operation> <deque> [<item>]

Description:
  Perform basic deque (double ended queue) operations from the cli

Available Flags:
  -debug
    	Enable debug mode
  -dry
    	Don't update file after change
  -file string
    	File to store deques in (default "~/.deque.json")
  -version
    	Print version info

Positional Arguments (argv):
  operation    Operation to perform on deque (required).
  deque        Deque to perform operation on (required, except keys).
  item         Item to prepend / append (required for unshift, push).

Available Operations:
  shift        Remove item from front of deque
  unshift      Add item to front of deque
  pop          Remove item from back of deque
  push         Add item to back of deque
  keys         List all deques
  list         List all values in a deque
  delete       Delete a deque
```

## Examples

Add an item to a new deque (front):

```shell
deque unshift mydeque item1
```

Add another item to the same deque (back):

```shell
deque push mydeque item2
```

View your deques:

```shell
deque keys
```

List items in a deque:

```shell
deque list mydeque
```

Remove an item from the front of the deque:

```shell
deque shift mydeque
```

Remove an item from the back of the deque:

```shell
deque pop mydeque
```

View an item without removing it from a deque:
```shell
deque -dry shift mydeque  # front
deque -dry pop mydeque    # back
```

