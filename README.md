# Deque

A double ended queue cli utility written in Go.

## Install

Tbd...

## Usage

```
Usage: deque [options] <operation> <queue> [<data>]

Description:
  Preform basic dequeue (double ended queue) operations from the cli

Available Flags:
  -file string
    	File to store dequeues in (default "~/.dq.json")

Positional Arguments (argv):
  operation    Operation to preform on queue (required).
  queue        Queue to perform operation on (required).
  data         Data to prepend / append (required for unshift, push).

Available Operations:
  shift        Remove item from front of queue
  unshift      Add item to front of queue
  pop          Remove item from back of queue
  push         Add item to back of queue
  keys         List all queues
  list         List all values in queue
  delete       Delete a queue
```
