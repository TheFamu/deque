# Deque

[![Test](https://github.com/BlueSquare23/deque/actions/workflows/actions.yml/badge.svg?branch=main)](https://github.com/BlueSquare23/deque/actions/workflows/actions.yml)

A double ended queue cli utility written in Go.

## Install

Tbd...

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
    	File to store deques in (default "/home/blue/.deque.json")

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
